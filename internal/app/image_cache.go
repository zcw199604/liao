package app

import (
	"sync"
	"time"
)

type ImageCacheService struct {
	mu    sync.Mutex
	cache map[string]*CachedImages
}

type CachedImages struct {
	ImageURLs  []string
	ExpireTime int64
}

func (c *CachedImages) isExpired() bool {
	return time.Now().UnixMilli() > c.ExpireTime
}

func NewImageCacheService() *ImageCacheService {
	return &ImageCacheService{
		cache: make(map[string]*CachedImages),
	}
}

const imageCacheExpireMs int64 = 3 * 60 * 60 * 1000

func (s *ImageCacheService) AddImageToCache(userID, imageURL string) {
	expire := time.Now().UnixMilli() + imageCacheExpireMs

	s.mu.Lock()
	defer s.mu.Unlock()

	cached := s.cache[userID]
	if cached == nil || cached.isExpired() {
		s.cache[userID] = &CachedImages{
			ImageURLs:  []string{imageURL},
			ExpireTime: expire,
		}
		return
	}

	cached.ImageURLs = append(cached.ImageURLs, imageURL)
	cached.ExpireTime = expire
}

func (s *ImageCacheService) GetCachedImages(userID string) *CachedImages {
	s.mu.Lock()
	defer s.mu.Unlock()

	cached := s.cache[userID]
	if cached == nil {
		return nil
	}
	if cached.isExpired() {
		delete(s.cache, userID)
		return nil
	}
	return &CachedImages{
		ImageURLs:  append([]string(nil), cached.ImageURLs...),
		ExpireTime: cached.ExpireTime,
	}
}

func (s *ImageCacheService) RebuildCache(userID string, imagePaths []string) {
	if len(imagePaths) == 0 {
		return
	}

	expire := time.Now().UnixMilli() + imageCacheExpireMs

	s.mu.Lock()
	defer s.mu.Unlock()

	cp := append([]string(nil), imagePaths...)
	s.cache[userID] = &CachedImages{ImageURLs: cp, ExpireTime: expire}
}

func (s *ImageCacheService) ClearCache(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cache, userID)
}
