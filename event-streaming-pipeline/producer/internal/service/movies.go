package service

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"producer/internal/config"
	"producer/internal/gen/api"
	repo "producer/internal/repository/kafka"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type MoviesService struct {
	p *repo.MoviesProducer
}

func New(cfg config.Config) (*MoviesService, error) {
	p, err := repo.New(cfg.Topic, cfg.KafkaServers, cfg.SchemaRegistryURL)
	return &MoviesService{p: p}, err
}

func (s *MoviesService) PostEvent(ctx context.Context, event api.MovieEvent) error {
	if err := s.p.PostEvent(ctx, event); err != nil {
		return err
	}
	log.Printf(
		"published event_id: %s event_type: %s time: %s",
		event.EventId.String(),
		event.EventType,
		event.Timestamp.String(),
	)
	return nil
}

func (s *MoviesService) GenerateEvents(ctx context.Context, sessions int) error {
	users := make([]string, 0, max(1, sessions/10))
	movies := make([]string, 0, max(1, sessions/5))
	devices := []api.DeviceType{api.DESKTOP, api.MOBILE, api.TABLET, api.TV}
	otherActions := []api.EventType{api.LIKED, api.SEARCHED}
	for range cap(users) {
		users = append(users, uuid.New().String())
	}
	for range cap(movies) {
		movies = append(movies, uuid.New().String())
	}
	g := &errgroup.Group{}
	for range sessions {
		g.Go(func() error {
			actions := make([]api.MovieEvent, 0, 2)
			userID := users[rand.Intn(len(users))]
			movieID := movies[rand.Intn(len(movies))]
			sessionID := uuid.New().String()
			device := devices[rand.Intn(len(devices))]
			ts := time.Now().UTC().Add(-time.Duration(rand.Intn(24*60*60*10)) * time.Second)
			actions = append(actions, api.MovieEvent{
				EventId:         uuid.New(),
				UserId:          userID,
				MovieId:         movieID,
				EventType:       api.VIEWSTARTED,
				Timestamp:       ts,
				DeviceType:      device,
				SessionId:       sessionID,
				ProgressSeconds: 0,
			})
			for range rand.Intn(3) {
				ts = ts.Add(time.Duration(5*60+rand.Intn(30*60)) * time.Second)
				actions = append(actions, api.MovieEvent{
					EventId:         uuid.New(),
					UserId:          userID,
					MovieId:         movieID,
					EventType:       api.VIEWPAUSED,
					Timestamp:       ts,
					DeviceType:      device,
					SessionId:       sessionID,
					ProgressSeconds: int(ts.Sub(actions[len(actions)-1].Timestamp).Seconds()) + actions[len(actions)-1].ProgressSeconds,
				})
				ts = ts.Add(time.Duration(5+rand.Intn(10*60)) * time.Second)
				actions = append(actions, api.MovieEvent{
					EventId:         uuid.New(),
					UserId:          userID,
					MovieId:         movieID,
					EventType:       api.VIEWRESUMED,
					Timestamp:       ts,
					DeviceType:      device,
					SessionId:       sessionID,
					ProgressSeconds: actions[len(actions)-1].ProgressSeconds,
				})
			}
			ts = ts.Add(time.Duration(15*60+rand.Intn(60*60)) * time.Second)
			actions = append(actions, api.MovieEvent{
				EventId:         uuid.New(),
				UserId:          userID,
				MovieId:         movieID,
				EventType:       api.VIEWFINISHED,
				Timestamp:       ts,
				DeviceType:      device,
				SessionId:       sessionID,
				ProgressSeconds: int(ts.Sub(actions[len(actions)-1].Timestamp).Seconds()) + actions[len(actions)-1].ProgressSeconds,
			})
			for _, a := range actions {
				err := s.PostEvent(ctx, a)
				if err != nil {
					return fmt.Errorf("%w | %s", err, a.EventType)
				}
			}
			return nil
		})
	}
	for range sessions {
		g.Go(func() error {
			userID := users[rand.Intn(len(users))]
			movieID := movies[rand.Intn(len(movies))]
			device := devices[rand.Intn(len(devices))]
			sessionID := uuid.New().String()
			ts := time.Now().UTC().Add(-time.Duration(rand.Intn(24*60*60*10)) * time.Second)
			action := api.MovieEvent{
				EventId:         uuid.New(),
				UserId:          userID,
				MovieId:         movieID,
				EventType:       otherActions[rand.Intn(len(otherActions))],
				Timestamp:       ts,
				DeviceType:      device,
				SessionId:       sessionID,
				ProgressSeconds: 0,
			}
			if err := s.PostEvent(ctx, action); err != nil {
				return fmt.Errorf("%w | %s", err, action.EventType)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}
