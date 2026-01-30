package shortener

import (
	"context"
	"sync"
	"time"

	"github.com/IvanOplesnin/url-shortener/internal/logger"
	"github.com/IvanOplesnin/url-shortener/internal/repository"
)

const (
	maxBuffer     = 1000
	flushInterval = 2 * time.Second
	queueCapacity = maxBuffer
)

type DeleteData struct {
	userID    int64
	shortURLs []string
}

func NewDeleteData(userID int64, shortURLs []string) DeleteData {
	return DeleteData{
		userID:    userID,
		shortURLs: shortURLs,
	}
}

type SoftDeleterService struct {
	reqCh chan DeleteData
	repo  repository.MarkUserDeleter

	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
	wg        sync.WaitGroup

	buffer  map[int64][]string
	counter uint64
}

func NewDeleterService(repo repository.MarkUserDeleter) *SoftDeleterService {
	return &SoftDeleterService{
		reqCh:  make(chan DeleteData, queueCapacity),
		buffer: make(map[int64][]string),
		repo:   repo,
		stopCh: make(chan struct{}),
	}
}

func (sd *SoftDeleterService) Start() {
	sd.startOnce.Do(func() {
		sd.wg.Add(1)
		go sd.loop()
	})
}

func (sd *SoftDeleterService) Stop() {
	sd.stopOnce.Do(func() {
		close(sd.stopCh)
	})
	sd.wg.Wait()
}

func (sd *SoftDeleterService) loop() {
	defer sd.wg.Done()

	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case data := <-sd.reqCh:
			if _, ok := sd.buffer[data.userID]; !ok {
				sd.buffer[data.userID] = data.shortURLs
			} else {
				sd.buffer[data.userID] = append(sd.buffer[data.userID], data.shortURLs...)
			}
			sd.counter += uint64((len(data.shortURLs)))
			if sd.counter >= maxBuffer {
				sd.flush()
			}
		case <-ticker.C:
			sd.flush()
		case <-sd.stopCh:
			sd.flush()
			return
		}
	}
}

func (sd *SoftDeleterService) flush() {
	if len(sd.buffer) == 0 {
		return
	}
	for userID, shortURLs := range sd.buffer {
		ctx := context.Background()
		if err := sd.repo.DeletedBatch(ctx, userID, shortURLs); err != nil {
			logger.Log.Errorf("error flushing buffer: %v", err)
		}
	}
	sd.buffer = make(map[int64][]string)
	sd.counter = 0
}

func (sd *SoftDeleterService) Add(data DeleteData) bool {
	select {
	case sd.reqCh <- data:
		return true
	default:
		return false
	}
}
