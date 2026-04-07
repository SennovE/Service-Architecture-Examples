package provider

import (
	"context"
	"errors"
	pb "flights/internal/gen/proto"
	"flights/internal/service"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type FlightsProvider struct {
	pb.UnimplementedFlightsServer
	srvc *service.FlightsService
	cache *Cache
}

func NewServer(srvc *service.FlightsService, rdb *redis.Client) pb.FlightsServer {
	return &FlightsProvider{srvc: srvc, cache: &Cache{rdb: rdb}}
}

func (p *FlightsProvider) GetFlight(
	ctx context.Context, request *pb.GetFlightRequest) (*pb.FlightResponse, error) {
	flightID, err := uuid.Parse(request.GetFlightId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid flight ID")
	}
	flight := p.cache.GetFlight(ctx, flightID)
	if flight != nil {
		return mapServiceToProvider(flight), nil
	}
	flight, err = p.srvc.GetFlight(ctx, flightID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFlightNotFound):
			return nil, status.Error(codes.NotFound, "flight not found")
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}
	go p.cache.SetFlight(flight, 5 * time.Minute)
	return mapServiceToProvider(flight), nil
}

func (p *FlightsProvider) ReleaseReservation(
	ctx context.Context, request *pb.ReleaseReservationRequest) (*pb.ReleaseReservationResponse, error) {
	bookingID, err := uuid.Parse(request.GetBookingId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid booking ID")
	}
	flightID, version, err := p.srvc.ReleaseReservation(ctx, bookingID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrBookingNotFound):
			return nil, status.Error(codes.NotFound, "booking not found")
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}
	go p.cache.RemoveFlight(flightID, version, 5 * time.Minute)
	return &pb.ReleaseReservationResponse{}, nil
}

func (p *FlightsProvider) ReserveSeats(
	ctx context.Context, request *pb.ReserveSeatsRequest) (*pb.ReserveSeatsResponse, error) {
	flightID, err := uuid.Parse(request.GetFlightId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid flight ID")
	}
	bookingID, err := uuid.Parse(request.GetBookingId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid booking ID")
	}
	price, version, err := p.srvc.ReserveSeats(
		ctx,
		flightID,
		bookingID,
		request.GetSeatCount(),
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFlightNotFound):
			return nil, status.Error(codes.NotFound, "flight not found")
		case errors.Is(err, service.ErrNotEnoughSeats):
			return nil, status.Error(codes.ResourceExhausted, "not enough seats")
		case errors.Is(err, service.ErrBookingExists):
			return nil, status.Error(codes.AlreadyExists, "booking already exists")
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}
	go p.cache.RemoveFlight(flightID, version, 5 * time.Minute)
	resp := &pb.ReserveSeatsResponse{}
	resp.SetPrice(price.String())
	return resp, nil
}

func (p *FlightsProvider) SearchFlights(
	request *pb.SearchFlightsRequest, stream grpc.ServerStreamingServer[pb.FlightResponse]) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var t *time.Time
	if request.HasDepartureDate() {
		tmp := request.GetDepartureDate().AsTime()
		t = &tmp
	}
	it := p.cache.GetSearchFlights(
		ctx,
		request.GetOrigin(),
		request.GetDestination(),
		t,
	)
	if it != nil {
		for flight := range it {
			resp := mapServiceToProvider(flight)
			if err := stream.Send(resp); err != nil {
				return err
			}
		}
		return nil
	}

	it, err := p.srvc.SearchFlights(
		ctx,
		request.GetOrigin(),
		request.GetDestination(),
		t,
	)
	if err != nil {
		return err
	}
	idsChan := make(chan *service.Flight)
	go p.cache.SetSearchFlights(request.GetOrigin(), request.GetDestination(), t, idsChan, 5 * time.Minute)
	for flight := range it {
		resp := mapServiceToProvider(flight)
		idsChan <- flight
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	close(idsChan)
	return nil
}

func mapServiceToProvider(flight *service.Flight) *pb.FlightResponse {
	resp := &pb.FlightResponse{}

	resp.SetId(flight.ID.String())
	resp.SetNum(flight.Num)
	resp.SetCompany(flight.Company)
	resp.SetIata(flight.IATA)
	resp.SetDepartureAirport(flight.DepartureAirport)
	resp.SetDestinationAirport(flight.DestinationAirport)
	resp.SetTotalSeats(flight.TotalSeats)
	resp.SetAvailableSeats(flight.AvailableSeats)
	resp.SetDepartureDate(timestamppb.New(flight.DepartureDate))
	resp.SetDestinationDate(timestamppb.New(flight.DestinationDate))
	resp.SetPrice(flight.Price.String())
	resp.SetStatus(pb.FlightStatus(pb.FlightStatus_value[flight.Status]))

	return resp
}
