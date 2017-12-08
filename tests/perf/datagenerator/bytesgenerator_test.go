package datagenerator

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/mock/gomock"
)

func TestGetBytes(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockDistribution := helpers.NewMockDistribution(mockCtrl)

	mockDistribution.EXPECT().Max().Return(100)
	bg := NewBytesGenerator(mockDistribution)

	mockDistribution.EXPECT().GetNext().Return(50)
	b1 := bg.GetBytes()
	if len(b1) != 50 {
		t.Errorf("expected bytes of length 50")
	}

	mockDistribution.EXPECT().GetNext().Return(75)
	b2 := bg.GetBytes()
	if len(b2) != 75 {
		t.Errorf("expected bytes of length 75")
	}

	// Make sure that b1 and b2 use the same memory address.
	hdr1 := (*reflect.SliceHeader)(unsafe.Pointer(&b1))
	hdr2 := (*reflect.SliceHeader)(unsafe.Pointer(&b2))
	if hdr1.Data != hdr2.Data {
		t.Errorf("expected b1 and b2 share the same address.")
	}
}
