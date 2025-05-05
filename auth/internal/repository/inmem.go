package repository

import (
	"fmt"
	"github.com/google/uuid"
	"sync"
)

type Post struct {
	UserId string
	Post   string
}

type InMem struct {
	Post map[string]*Post
	Mu   *sync.Mutex
}

func (i *InMem) Get(key, userId string) (*Post, error) {
	i.Mu.Lock()
	defer i.Mu.Unlock()

	if _, ok := i.Post[key]; !ok {
		return nil, fmt.Errorf("key not found")
	}

	if i.Post[key].UserId != userId {
		return nil, fmt.Errorf("invalid")
	}

	return i.Post[key], nil
}

func (i *InMem) GetAll(userId string) ([]*Post, error) {
	i.Mu.Lock()
	defer i.Mu.Unlock()

	var posts []*Post
	for _, post := range i.Post {
		if post.UserId == userId {
			posts = append(posts, post)
		}
	}

	if len(posts) == 0 {
		return nil, fmt.Errorf("no posts found")
	}

	return posts, nil
}

func (i *InMem) Set(userId, value string) error {
	i.Mu.Lock()
	defer i.Mu.Unlock()

	id := uuid.NewString()

	i.Post[id] = &Post{
		UserId: userId,
		Post:   value,
	}

	return nil
}

func (i *InMem) Delete(key string, userId string) error {
	i.Mu.Lock()
	defer i.Mu.Unlock()

	if _, ok := i.Post[key]; !ok {
		return fmt.Errorf("key not found")
	}

	if i.Post[key].UserId != userId {
		return fmt.Errorf("invalid")
	}

	delete(i.Post, key)
	return nil
}
