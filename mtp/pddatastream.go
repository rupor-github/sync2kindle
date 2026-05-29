//go:build mtp && windows

package mtp

import (
	"unsafe"

	ole "github.com/go-ole/go-ole"
)

type IPortableDeviceDataStream struct {
	ole.IUnknown
}

type IPortableDeviceDataStreamVtbl struct {
	ole.IUnknownVtbl
	Read         uintptr // ISequentialStreamVtbl
	Write        uintptr
	Seek         uintptr // IStreamVtbl
	SetSize      uintptr
	CopyTo       uintptr
	Commit       uintptr
	Revert       uintptr
	LockRegion   uintptr
	UnlockRegion uintptr
	Stat         uintptr
	Clone        uintptr
	GetObjectID  uintptr // IPortableDeviceDataStreamVtbl
	Cancel       uintptr
}

func (v *IPortableDeviceDataStream) VTable() *IPortableDeviceDataStreamVtbl {
	return (*IPortableDeviceDataStreamVtbl)(unsafe.Pointer(v.RawVTable))
}

//---------------------------------------------------------
// PortableDeviceApi.idl
//---------------------------------------------------------
// This interface can be used to read/write data during transfers.
// It is retrieved by doing a QI on the IStream for a resource
// data or object creation request.
// It has several PortableDeviceApi specific methods in
// addition to the ones provided by IStream.
//---------------------------------------------------------
// [
//     object,
//     uuid(88e04db3-1012-4d64-9996-f703a950d3f4),
//     helpstring("IPortableDeviceDataStream Interface"),
//     pointer_default(unique)
// ]
// interface IPortableDeviceDataStream : IStream
// {
//     HRESULT GetObjectID(
//         [out] LPWSTR* ppszObjectID);
//
//     HRESULT Cancel();
// };
//---------------------------------------------------------
// objidlbase.idl
//---------------------------------------------------------
// [
//     object,
//     uuid(0000000c-0000-0000-C000-000000000046),
//     pointer_default(unique)
// ]
//
// interface IStream : ISequentialStream
// {
//
//     typedef [unique] IStream *LPSTREAM;
//
//     /* Storage stat buffer */
//
//     typedef struct tagSTATSTG
//     {
//         LPOLESTR pwcsName;
//         DWORD type;
//         ULARGE_INTEGER cbSize;
//         FILETIME mtime;
//         FILETIME ctime;
//         FILETIME atime;
//         DWORD grfMode;
//         DWORD grfLocksSupported;
//         CLSID clsid;
//         DWORD grfStateBits;
//     DWORD reserved;
//     } STATSTG;
//
//
//     /* Storage element types */
//     typedef enum tagSTGTY
//     {
//         STGTY_STORAGE   = 1,
//         STGTY_STREAM    = 2,
//         STGTY_LOCKBYTES = 3,
//         STGTY_PROPERTY  = 4
//     } STGTY;
//
//     typedef enum tagSTREAM_SEEK
//     {
//         STREAM_SEEK_SET = 0,
//         STREAM_SEEK_CUR = 1,
//         STREAM_SEEK_END = 2
//     } STREAM_SEEK;
//
//     typedef enum tagLOCKTYPE
//     {
//         LOCK_WRITE      = 1,
//         LOCK_EXCLUSIVE  = 2,
//         LOCK_ONLYONCE   = 4
//     } LOCKTYPE;
//
//     HRESULT Seek(
//         [in] LARGE_INTEGER dlibMove,
//         [in] DWORD dwOrigin,
//         [annotation("_Out_opt_")] ULARGE_INTEGER *plibNewPosition);
//
//     HRESULT SetSize(
//         [in] ULARGE_INTEGER libNewSize);
//
//     HRESULT CopyTo(
//         [in, unique, annotation("_In_")] IStream *pstm,
//         [in] ULARGE_INTEGER cb,
//         [annotation("_Out_opt_")] ULARGE_INTEGER *pcbRead,
//         [annotation("_Out_opt_")] ULARGE_INTEGER *pcbWritten);
//
//     HRESULT Commit(
//         [in] DWORD grfCommitFlags);
//
//     HRESULT Revert();
//
//     HRESULT LockRegion(
//         [in] ULARGE_INTEGER libOffset,
//         [in] ULARGE_INTEGER cb,
//         [in] DWORD dwLockType);
//
//     HRESULT UnlockRegion(
//         [in] ULARGE_INTEGER libOffset,
//         [in] ULARGE_INTEGER cb,
//         [in] DWORD dwLockType);
//
//     HRESULT Stat(
//         [out] STATSTG *pstatstg,
//         [in] DWORD grfStatFlag);
//
//     HRESULT Clone(
//         [out] IStream **ppstm);
// }
//---------------------------------------------------------
// [
//     object,
//     uuid(0c733a30-2a1c-11ce-ade5-00aa0044773d),
//     pointer_default(unique)
// ]
// interface ISequentialStream : IUnknown
// {
//     HRESULT Read(
//         [annotation("_Out_writes_bytes_to_(cb, *pcbRead)")]
//         void *pv,
//         [in, annotation("_In_")] ULONG cb,
//         [annotation("_Out_opt_")] ULONG *pcbRead);
//
//     HRESULT Write(
//         [annotation("_In_reads_bytes_(cb)")] void const *pv,
//         [in, annotation("_In_")] ULONG cb,
//         [annotation("_Out_opt_")] ULONG *pcbWritten);
// }
