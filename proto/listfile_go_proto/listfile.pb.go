// Code generated by protoc-gen-go. DO NOT EDIT.
// source: listfile.proto

package listfile_go_proto

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

// List File Entry specification.
type ListFileEntry struct {
	// Types that are valid to be assigned to Entry:
	//	*ListFileEntry_FileInfo
	//	*ListFileEntry_DirectoryInfo
	//	*ListFileEntry_DirectoryHeader
	Entry                isListFileEntry_Entry `protobuf_oneof:"entry"`
	XXX_NoUnkeyedLiteral struct{}              `json:"-"`
	XXX_unrecognized     []byte                `json:"-"`
	XXX_sizecache        int32                 `json:"-"`
}

func (m *ListFileEntry) Reset()         { *m = ListFileEntry{} }
func (m *ListFileEntry) String() string { return proto.CompactTextString(m) }
func (*ListFileEntry) ProtoMessage()    {}
func (*ListFileEntry) Descriptor() ([]byte, []int) {
	return fileDescriptor_944e22c88393983d, []int{0}
}

func (m *ListFileEntry) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListFileEntry.Unmarshal(m, b)
}
func (m *ListFileEntry) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListFileEntry.Marshal(b, m, deterministic)
}
func (m *ListFileEntry) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListFileEntry.Merge(m, src)
}
func (m *ListFileEntry) XXX_Size() int {
	return xxx_messageInfo_ListFileEntry.Size(m)
}
func (m *ListFileEntry) XXX_DiscardUnknown() {
	xxx_messageInfo_ListFileEntry.DiscardUnknown(m)
}

var xxx_messageInfo_ListFileEntry proto.InternalMessageInfo

type isListFileEntry_Entry interface {
	isListFileEntry_Entry()
}

type ListFileEntry_FileInfo struct {
	FileInfo *FileInfo `protobuf:"bytes,1,opt,name=file_info,json=fileInfo,proto3,oneof"`
}

type ListFileEntry_DirectoryInfo struct {
	DirectoryInfo *DirectoryInfo `protobuf:"bytes,2,opt,name=directory_info,json=directoryInfo,proto3,oneof"`
}

type ListFileEntry_DirectoryHeader struct {
	DirectoryHeader *DirectoryHeader `protobuf:"bytes,3,opt,name=directory_header,json=directoryHeader,proto3,oneof"`
}

func (*ListFileEntry_FileInfo) isListFileEntry_Entry() {}

func (*ListFileEntry_DirectoryInfo) isListFileEntry_Entry() {}

func (*ListFileEntry_DirectoryHeader) isListFileEntry_Entry() {}

func (m *ListFileEntry) GetEntry() isListFileEntry_Entry {
	if m != nil {
		return m.Entry
	}
	return nil
}

func (m *ListFileEntry) GetFileInfo() *FileInfo {
	if x, ok := m.GetEntry().(*ListFileEntry_FileInfo); ok {
		return x.FileInfo
	}
	return nil
}

func (m *ListFileEntry) GetDirectoryInfo() *DirectoryInfo {
	if x, ok := m.GetEntry().(*ListFileEntry_DirectoryInfo); ok {
		return x.DirectoryInfo
	}
	return nil
}

func (m *ListFileEntry) GetDirectoryHeader() *DirectoryHeader {
	if x, ok := m.GetEntry().(*ListFileEntry_DirectoryHeader); ok {
		return x.DirectoryHeader
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*ListFileEntry) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*ListFileEntry_FileInfo)(nil),
		(*ListFileEntry_DirectoryInfo)(nil),
		(*ListFileEntry_DirectoryHeader)(nil),
	}
}

// Represents a single file’s metadata.
type FileInfo struct {
	// Full path of the file in the format used by the local OS.
	Path string `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	// Last modified time of the file in seconds since the epoch.
	LastModifiedTime int64 `protobuf:"varint,2,opt,name=last_modified_time,json=lastModifiedTime,proto3" json:"last_modified_time,omitempty"`
	// The size of the file in bytes.
	Size                 int64    `protobuf:"varint,3,opt,name=size,proto3" json:"size,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *FileInfo) Reset()         { *m = FileInfo{} }
func (m *FileInfo) String() string { return proto.CompactTextString(m) }
func (*FileInfo) ProtoMessage()    {}
func (*FileInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_944e22c88393983d, []int{1}
}

func (m *FileInfo) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_FileInfo.Unmarshal(m, b)
}
func (m *FileInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_FileInfo.Marshal(b, m, deterministic)
}
func (m *FileInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_FileInfo.Merge(m, src)
}
func (m *FileInfo) XXX_Size() int {
	return xxx_messageInfo_FileInfo.Size(m)
}
func (m *FileInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_FileInfo.DiscardUnknown(m)
}

var xxx_messageInfo_FileInfo proto.InternalMessageInfo

func (m *FileInfo) GetPath() string {
	if m != nil {
		return m.Path
	}
	return ""
}

func (m *FileInfo) GetLastModifiedTime() int64 {
	if m != nil {
		return m.LastModifiedTime
	}
	return 0
}

func (m *FileInfo) GetSize() int64 {
	if m != nil {
		return m.Size
	}
	return 0
}

// Represents a single directory's metadata.
type DirectoryInfo struct {
	// The full path of the directory in the format used by the local OS.
	Path                 string   `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DirectoryInfo) Reset()         { *m = DirectoryInfo{} }
func (m *DirectoryInfo) String() string { return proto.CompactTextString(m) }
func (*DirectoryInfo) ProtoMessage()    {}
func (*DirectoryInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_944e22c88393983d, []int{2}
}

func (m *DirectoryInfo) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DirectoryInfo.Unmarshal(m, b)
}
func (m *DirectoryInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DirectoryInfo.Marshal(b, m, deterministic)
}
func (m *DirectoryInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DirectoryInfo.Merge(m, src)
}
func (m *DirectoryInfo) XXX_Size() int {
	return xxx_messageInfo_DirectoryInfo.Size(m)
}
func (m *DirectoryInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_DirectoryInfo.DiscardUnknown(m)
}

var xxx_messageInfo_DirectoryInfo proto.InternalMessageInfo

func (m *DirectoryInfo) GetPath() string {
	if m != nil {
		return m.Path
	}
	return ""
}

// Contains information about the directory that's being listed.
// The contents (files and directories) of the directory will appear below
// the DirectoryHeader in the list file.
type DirectoryHeader struct {
	// The full path of the directory in the format used by the local OS.
	Path string `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	// The number of list file entries, each representing a file or directory
	// present in the directory specified by path, that follow this header.
	NumEntries           int64    `protobuf:"varint,2,opt,name=num_entries,json=numEntries,proto3" json:"num_entries,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DirectoryHeader) Reset()         { *m = DirectoryHeader{} }
func (m *DirectoryHeader) String() string { return proto.CompactTextString(m) }
func (*DirectoryHeader) ProtoMessage()    {}
func (*DirectoryHeader) Descriptor() ([]byte, []int) {
	return fileDescriptor_944e22c88393983d, []int{3}
}

func (m *DirectoryHeader) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DirectoryHeader.Unmarshal(m, b)
}
func (m *DirectoryHeader) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DirectoryHeader.Marshal(b, m, deterministic)
}
func (m *DirectoryHeader) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DirectoryHeader.Merge(m, src)
}
func (m *DirectoryHeader) XXX_Size() int {
	return xxx_messageInfo_DirectoryHeader.Size(m)
}
func (m *DirectoryHeader) XXX_DiscardUnknown() {
	xxx_messageInfo_DirectoryHeader.DiscardUnknown(m)
}

var xxx_messageInfo_DirectoryHeader proto.InternalMessageInfo

func (m *DirectoryHeader) GetPath() string {
	if m != nil {
		return m.Path
	}
	return ""
}

func (m *DirectoryHeader) GetNumEntries() int64 {
	if m != nil {
		return m.NumEntries
	}
	return 0
}

func init() {
	proto.RegisterType((*ListFileEntry)(nil), "cloud_ingest_listfile.ListFileEntry")
	proto.RegisterType((*FileInfo)(nil), "cloud_ingest_listfile.FileInfo")
	proto.RegisterType((*DirectoryInfo)(nil), "cloud_ingest_listfile.DirectoryInfo")
	proto.RegisterType((*DirectoryHeader)(nil), "cloud_ingest_listfile.DirectoryHeader")
}

func init() { proto.RegisterFile("listfile.proto", fileDescriptor_944e22c88393983d) }

var fileDescriptor_944e22c88393983d = []byte{
	// 327 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x92, 0xcf, 0x4b, 0xfb, 0x40,
	0x10, 0xc5, 0xdb, 0x6f, 0xbf, 0x6a, 0x3b, 0xa5, 0x3f, 0x58, 0x10, 0x7a, 0xab, 0x44, 0x11, 0x0f,
	0x9a, 0x80, 0xde, 0x3d, 0xb4, 0xb6, 0x56, 0xb0, 0x20, 0xd1, 0x93, 0x97, 0x35, 0xed, 0x4e, 0xd2,
	0x81, 0xdd, 0x6c, 0x49, 0x36, 0x87, 0xfa, 0xb7, 0x7b, 0x90, 0xdd, 0x26, 0x68, 0xa5, 0xe2, 0x29,
	0xc3, 0xbc, 0xdd, 0x4f, 0xde, 0x7b, 0x2c, 0x74, 0x25, 0xe5, 0x26, 0x26, 0x89, 0xfe, 0x3a, 0xd3,
	0x46, 0xb3, 0xe3, 0xa5, 0xd4, 0x85, 0xe0, 0x94, 0x26, 0x98, 0x1b, 0x5e, 0x89, 0xde, 0x47, 0x1d,
	0x3a, 0x8f, 0x94, 0x9b, 0x29, 0x49, 0x9c, 0xa4, 0x26, 0xdb, 0xb0, 0x5b, 0x68, 0x59, 0x85, 0x53,
	0x1a, 0xeb, 0x41, 0xfd, 0xa4, 0x7e, 0xd1, 0xbe, 0x1e, 0xfa, 0x7b, 0x2f, 0xfb, 0xf6, 0xd2, 0x43,
	0x1a, 0xeb, 0x59, 0x2d, 0x6c, 0xc6, 0xe5, 0xcc, 0xe6, 0xd0, 0x15, 0x94, 0xe1, 0xd2, 0xe8, 0x6c,
	0xb3, 0x85, 0xfc, 0x73, 0x90, 0xb3, 0x5f, 0x20, 0x77, 0xd5, 0xe1, 0x92, 0xd4, 0x11, 0xdf, 0x17,
	0xec, 0x19, 0xfa, 0x5f, 0xb8, 0x15, 0x46, 0x02, 0xb3, 0x41, 0xc3, 0x01, 0xcf, 0xff, 0x02, 0xce,
	0xdc, 0xe9, 0x59, 0x2d, 0xec, 0x89, 0xdd, 0xd5, 0xe8, 0x08, 0x0e, 0xd0, 0x86, 0xf5, 0xde, 0xa0,
	0x59, 0x85, 0x60, 0x0c, 0xfe, 0xaf, 0x23, 0xb3, 0x72, 0x99, 0x5b, 0xa1, 0x9b, 0xd9, 0x25, 0x30,
	0x19, 0xe5, 0x86, 0x2b, 0x2d, 0x28, 0x26, 0x14, 0xdc, 0x90, 0x42, 0x17, 0xa8, 0x11, 0xf6, 0xad,
	0x32, 0x2f, 0x85, 0x17, 0x52, 0x68, 0x09, 0x39, 0xbd, 0xa3, 0xf3, 0xd7, 0x08, 0xdd, 0xec, 0x9d,
	0x42, 0x67, 0x27, 0xe1, 0xbe, 0xdf, 0x78, 0x53, 0xe8, 0xfd, 0x70, 0xbd, 0xd7, 0xcd, 0x10, 0xda,
	0x69, 0xa1, 0xb8, 0xb5, 0x4e, 0x98, 0x97, 0x36, 0x20, 0x2d, 0xd4, 0x64, 0xbb, 0x19, 0x4d, 0x5e,
	0xc7, 0x09, 0x99, 0x55, 0xb1, 0xf0, 0x97, 0x5a, 0x05, 0xf7, 0x5a, 0x27, 0x12, 0xc7, 0xb6, 0xa4,
	0x27, 0x19, 0x99, 0x58, 0x67, 0x2a, 0x70, 0x95, 0x5d, 0x6d, 0x2b, 0x0b, 0xdc, 0xcb, 0x08, 0xaa,
	0xe2, 0x78, 0xa2, 0xb9, 0xdb, 0x2c, 0x0e, 0xdd, 0xe7, 0xe6, 0x33, 0x00, 0x00, 0xff, 0xff, 0xac,
	0x0f, 0x80, 0xd6, 0x44, 0x02, 0x00, 0x00,
}
