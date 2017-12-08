package gcloud

import (
	"errors"
	"reflect"
	"testing"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

func TestNewObjectIterator_Empty(t *testing.T) {
	iter := NewObjectIterator()

	if iter == nil {
		t.Error("got nil iterator, expected non-nil result")
	} else {
		_, err := iter.Next()
		if err != iterator.Done {
			t.Errorf("got error %v, expected %v", err, iterator.Done)
		}
	}
}

func TestNewObjectIterator_NonEmpty(t *testing.T) {
	attrs1 := &storage.ObjectAttrs{Generation: 123}
	attrs2 := &storage.ObjectAttrs{Generation: 234}

	iter := NewObjectIterator(attrs1, attrs2)

	if iter == nil {
		t.Error("got nil iterator, expected non-nil result")
	} else {
		for i, want := range []*storage.ObjectAttrs{attrs1, attrs2} {
			got, err := iter.Next()
			if err != nil {
				t.Errorf("got %v, expected no error, in iteration %d", err, i)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got %v, expected %v in iteration %d", got, want, i)
			}
		}

		_, err := iter.Next()
		if err != iterator.Done {
			t.Errorf("got error %v, expected %v", err, iterator.Done)
		}
	}
}

func TestNewObjectIterator_Error(t *testing.T) {
	want := errors.New("failed to iterate")
	iter := NewObjectIterator(want)

	if iter == nil {
		t.Error("got nil iterator, expected non-nil result")
	} else {
		_, err := iter.Next()
		if err != want {
			t.Errorf("got error %v, expected %v", err, want)
		}
	}
}
