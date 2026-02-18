package mtp

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	ole "github.com/go-ole/go-ole"
	"go.uber.org/zap"
	"golang.org/x/sys/windows"

	"s2k/common"
	"s2k/misc"
	"s2k/objects"
)

type Device struct {
	log            *zap.Logger
	id             common.PnPDeviceID
	pdmanager      *IPortableDeviceManager
	clientInfo     *IPortableDeviceValues
	pdevice        *IPortableDevice
	availableBytes int64
	freeBytes      int64
	fullAccess     bool
	storage        string
	roots          []string
}

// Connect to the supported device.
func Connect(paths, serial string, _ bool, log *zap.Logger) (d *Device, err error) {
	defer func() {
		if err != nil {
			d.Disconnect()
			d = nil
		}
	}()
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		return nil, err
	}
	d = &Device{log: log.Named(driverName)}
	d.pdmanager, err = CreatePortableDeviceManager()
	if err != nil {
		return
	}
	d.id, err = pickDevice(d.pdmanager, serial, log)
	if err != nil {
		return
	}
	d.clientInfo, err = prepareClientInfo()
	if err != nil {
		return
	}
	d.pdevice, err = CreatePortableDevice()
	if err != nil {
		return
	}
	err = d.pdevice.Open(d.id, d.clientInfo)
	if err != nil {
		err = fmt.Errorf("failed to Open device '%s': %w", d.id, err)
		return
	}
	info, err := d.fillDeviceInfo()
	if err != nil {
		err = fmt.Errorf("failed to get device info properties: %w", err)
		return
	}
	d.log.Debug("Device Info", zap.Any("Properties", info))
	storage, err := d.fillStorageInfo()
	if err != nil {
		err = fmt.Errorf("failed to get device storage properties: %w", err)
		return
	}
	d.log.Debug("Device Storage", zap.Any("Properties", storage))
	if !d.fullAccess {
		err = common.ErrNoAccess
	}

	// prepare filters for the device - has to be after we know selected storage root
	ps := filepath.SplitList(paths)
	for _, p := range ps {
		p = path.Join(d.storage, p)
		if !slices.Contains(d.roots, p) {
			d.roots = append(d.roots, p)
		}
	}
	d.log.Debug("Device paths of interest", zap.Any("Roots", d.roots))
	return
}

// driver interface

func (d *Device) Disconnect() {
	if d == nil {
		return
	}
	if d.pdevice != nil {
		d.pdevice.Release()
		d.pdevice = nil
	}
	if d.clientInfo != nil {
		d.clientInfo.Release()
		d.clientInfo = nil
	}
	if d.pdmanager != nil {
		d.pdmanager.Release()
		d.pdmanager = nil
	}
	ole.CoUninitialize()
}

func (d *Device) UniqueID() string {
	return d.id.Serial()
}

func (d *Device) Name() string {
	return driverName
}

func (d *Device) MkDir(obj *objects.ObjectInfo) (err error) {
	if obj == nil {
		panic("MkDir is called with nil object")
	}
	parent := obj.OIS.Find(path.Dir(obj.FullPath))
	if parent == nil {
		return fmt.Errorf("parent object not found for '%s'", obj.FullPath)
	}
	obj.OidParent = parent.Oid

	defer func(start time.Time) {
		d.log.Debug("Executed action MkDir", zap.String("actor", d.Name()), zap.Any("object", obj), zap.Duration("elapsed", time.Since(start)), zap.Error(err))
	}(time.Now())

	values, err := createObjectValues(obj.OidParent, obj.Name, &WPD_CONTENT_TYPE_FOLDER, 0)
	if err != nil {
		return fmt.Errorf("failed to create object values: %w", err)
	}
	defer values.Release()

	content, err := d.pdevice.Content()
	if err != nil {
		return fmt.Errorf("failed to get device Content: %w", err)
	}
	defer content.Release()

	obj.Oid, err = content.CreateObjectWithPropertiesOnly(values)
	if err != nil {
		return fmt.Errorf("failed to CreateObjectWithPropertiesOnly for '%s': %w", obj.FullPath, err)
	}
	return nil
}

func (d *Device) Remove(obj *objects.ObjectInfo) (err error) {
	if obj == nil {
		panic("Remove is called with nil object")
	}

	defer func(start time.Time) {
		d.log.Debug("Executed action Remove", zap.String("actor", d.Name()), zap.Any("object", obj), zap.Duration("elapsed", time.Since(start)), zap.Error(err))
	}(time.Now())

	ids, err := CreatePortableDevicePropVariantCollection()
	if err != nil {
		return err
	}
	defer ids.Release()

	pv, err := NewPropVariantFromUTF16(obj.Oid)
	if err != nil {
		return fmt.Errorf("failed to create PROPVARIANT from string '%s': %w", obj.Oid.String(), err)
	}
	defer pv.Clear()

	if err := ids.Add(pv); err != nil {
		return fmt.Errorf("failed to Add PROPVARIANT to collection: %w", err)
	}

	content, err := d.pdevice.Content()
	if err != nil {
		return fmt.Errorf("failed to get device Content: %w", err)
	}
	defer content.Release()

	if err := content.Delete(DeviceDeleteNoRecursion, ids); err != nil {
		var oleerr *ole.OleError
		if errors.As(err, &oleerr) && oleerr.Code() == uintptr(windows.ERROR_INVALID_OPERATION) {
			return fmt.Errorf("failed to Delete object '%s' (has children): %w", obj.Oid, err)
		}
		return fmt.Errorf("failed to Delete object '%s': %w", obj.Oid, err)
	}
	return nil
}

func (d *Device) Copy(obj *objects.ObjectInfo) (err error) {
	if obj == nil {
		panic("Copy is called with nil object")
	}

	parent := obj.OIS.Find(path.Dir(obj.FullPath))
	if parent == nil {
		return fmt.Errorf("parent object not found for '%s'", obj.FullPath)
	}
	obj.OidParent = parent.Oid

	defer func(start time.Time) {
		d.log.Debug("Executed action Copy", zap.String("actor", d.Name()), zap.Any("object", obj), zap.Duration("elapsed", time.Since(start)), zap.Error(err))
	}(time.Now())

	from, err := os.Open(obj.ObjectName)
	if err != nil {
		return fmt.Errorf("unable to open source file '%s': %w", obj.ObjectName, err)
	}
	defer from.Close()

	values, err := createObjectValues(obj.OidParent, obj.Name, &WPD_CONTENT_TYPE_GENERIC_FILE, obj.ObjSize)
	if err != nil {
		return fmt.Errorf("failed to create object values: %w", err)
	}
	defer values.Release()

	content, err := d.pdevice.Content()
	if err != nil {
		return fmt.Errorf("failed to get device Content: %w", err)
	}
	defer content.Release()

	stream, bufsize, err := content.CreateObjectWithPropertiesAndData(values)
	if err != nil {
		return fmt.Errorf("failed to CreateObjectWithPropertiesAndData for '%s': %w", obj.FullPath, err)
	}
	defer stream.Release()

	written, err := io.CopyBuffer(stream, from, make([]byte, bufsize))
	if err != nil {
		var oleerr *ole.OleError
		if errors.As(err, &oleerr) {
			switch windows.Handle(oleerr.Code()) {
			case windows.STG_E_MEDIUMFULL:
				err = fmt.Errorf("failed to Copy file '%s' to '%s', device storage does not have enough free space : %w", obj.ObjectName, obj.FullPath, err)
			case windows.STG_E_ACCESSDENIED:
				err = fmt.Errorf("failed to Copy file '%s' to '%s', access denied: %w", obj.ObjectName, obj.FullPath, err)
			case windows.STG_E_WRITEFAULT:
				err = fmt.Errorf("failed to Copy file '%s' to '%s', io error on device: %w", obj.ObjectName, obj.FullPath, err)
			default:
				err = fmt.Errorf("failed to Copy file '%s' to '%s': %w", obj.ObjectName, obj.FullPath, err)
			}
		} else {
			err = fmt.Errorf("failed to Copy file '%s' to '%s': %w", obj.ObjectName, obj.FullPath, err)
		}
		stream.Revert()
		return err
	}
	if written != obj.ObjSize {
		stream.Revert()
		return fmt.Errorf("failed to Copy file '%s' (%d) to '%s' (%d), not all bytes have been written", obj.ObjectName, obj.ObjSize, obj.FullPath, written)
	}
	if err := stream.Commit(STGCDefault); err != nil {
		return fmt.Errorf("failed to Copy file '%s' to '%s', unable to commit changes: %w", obj.ObjectName, obj.FullPath, err)
	}
	obj.Oid, err = stream.GetObjectID()
	if err != nil {
		return fmt.Errorf("failed to Copy file '%s' to '%s', unable to get new object id: %w", obj.ObjectName, obj.FullPath, err)
	}
	return nil
}

func (d *Device) GetObjectInfos() (objects.ObjectInfoSet, error) {
	content, err := d.pdevice.Content()
	if err != nil {
		return nil, fmt.Errorf("failed to get device Content: %w", err)
	}
	defer content.Release()
	properties, err := content.Properties()
	if err != nil {
		return nil, fmt.Errorf("failed to get content Properties: %w", err)
	}
	defer properties.Release()

	type kdef struct {
		value *PropertyKey
	}

	kdefsCommon := []kdef{
		{WPD_OBJECT_CONTENT_TYPE},
		{WPD_OBJECT_NAME},
		{WPD_OBJECT_PARENT_ID},
		{WPD_OBJECT_PERSISTENT_UNIQUE_ID},
	}
	var keysCommon *IPortableDeviceKeyCollection
	if keysCommon, err = CreatePortableDeviceKeyCollection(); err != nil {
		return nil, err
	}
	defer keysCommon.Release()
	for i, t := range kdefsCommon {
		if err := keysCommon.Add(t.value); err != nil {
			return nil, fmt.Errorf("failed to Add key '%d' to common keys : %w", i, err)
		}
	}

	kdefsObjects := []kdef{
		{WPD_OBJECT_ORIGINAL_FILE_NAME},
		{WPD_OBJECT_CAN_DELETE},
		{WPD_OBJECT_SIZE},
		{WPD_OBJECT_DATE_CREATED},
		{WPD_OBJECT_DATE_MODIFIED},
	}
	var keysObjects *IPortableDeviceKeyCollection
	if keysObjects, err = CreatePortableDeviceKeyCollection(); err != nil {
		return nil, err
	}
	defer keysObjects.Release()
	for i, t := range kdefsObjects {
		if err := keysObjects.Add(t.value); err != nil {
			return nil, fmt.Errorf("failed to Add key '%d' to file keys : %w", i, err)
		}
	}

	infos := d.enumerateObjects(WPD_DEVICE_OBJECT_ID, "", content, properties, keysCommon, keysObjects, make([]*objects.ObjectInfo, 0))

	// index the results by full path, loosing target directories
	oset := objects.New()
	for _, info := range infos {
		oset[info.FullPath] = info
	}
	return oset, nil
}

// implementation

func (d *Device) enumerateObjects(
	id objects.ObjectID, root string,
	content *IPortableDeviceContent,
	properties *IPortableDeviceProperties,
	keysCommon, keysObjects *IPortableDeviceKeyCollection,
	infos []*objects.ObjectInfo) []*objects.ObjectInfo {

	info, err := getObjectInfo(id, properties, keysCommon, keysObjects)
	if err != nil {
		d.log.Warn("Unable to get values for an object, ignoring", zap.String("base", root), zap.Stringer("obj", id), zap.Error(err))
	}

	realObj := info.Dir || info.File

	name := info.Name
	if !realObj {
		name = id.String()
	}

	fullPath := path.Join(root, name)

	cont := false
	for _, r := range d.roots {
		if strings.HasPrefix(fullPath, r) || strings.HasPrefix(r, fullPath) {
			cont = true
			break
		}
	}

	// to save time we only drill down and keep objects under paths of interest
	if !cont {
		return infos
	}

	if realObj {
		if exists := slices.ContainsFunc(infos, func(oi *objects.ObjectInfo) bool {
			return slices.Equal(oi.Oid, id)
		}); exists {
			d.log.Warn("Object already in map, ignoring", zap.String("root", root), zap.Stringer("obj", id))
		} else {
			info.FullPath = strings.TrimPrefix(fullPath, d.storage+"/")
			infos = append(infos, info)
		}
	}

	objects, err := content.EnumObjects(0, id, nil)
	if err != nil {
		d.log.Warn("EnumObjects failed, ignoring", zap.String("root", root), zap.Stringer("obj", id), zap.Error(err))
		return infos
	}
	defer objects.Release()

	for {
		oids, err := objects.Next(1)
		if err != nil {
			break
		}
		for _, oid := range oids {
			infos = d.enumerateObjects(oid, fullPath, content, properties, keysCommon, keysObjects, infos)
		}
	}
	return infos
}

func getObjectInfo(
	id objects.ObjectID,
	properties *IPortableDeviceProperties,
	keysCommon, keysObjects *IPortableDeviceKeyCollection) (*objects.ObjectInfo, error) {

	valuesCommon, err := properties.GetValues(id, keysCommon)
	if err != nil {
		return nil, fmt.Errorf("unable to get common values on object '%s': %w", id, err)
	}
	defer valuesCommon.Release()

	guid, err := valuesCommon.GetGuidValue(WPD_OBJECT_CONTENT_TYPE)
	if err != nil {
		return nil, fmt.Errorf("unable to get object content type: %w", err)
	}

	o := &objects.ObjectInfo{Oid: id}
	o.File = !ole.IsEqualGUID(&WPD_CONTENT_TYPE_FUNCTIONAL_OBJECT, &guid) && !ole.IsEqualGUID(&WPD_CONTENT_TYPE_FOLDER, &guid)
	o.Dir = !ole.IsEqualGUID(&WPD_CONTENT_TYPE_FUNCTIONAL_OBJECT, &guid) && ole.IsEqualGUID(&WPD_CONTENT_TYPE_FOLDER, &guid)

	parent, err := valuesCommon.GetStringValue(WPD_OBJECT_PARENT_ID)
	if err != nil {
		return o, fmt.Errorf("unable to get object parent id: %w", err)
	}
	o.OidParent, err = windows.UTF16FromString(parent)
	if err != nil {
		return o, fmt.Errorf("unable to convert parent id to UTF16: %w", err)
	}
	o.PersistentID, err = valuesCommon.GetStringValue(WPD_OBJECT_PERSISTENT_UNIQUE_ID)
	if err != nil {
		return o, fmt.Errorf("unable to get object persistent unique id: %w", err)
	}
	o.ObjectName, err = valuesCommon.GetStringValue(WPD_OBJECT_NAME)
	if err != nil {
		return o, fmt.Errorf("unable to get object name: %w", err)
	}

	if !o.File && !o.Dir {
		// we have a functional object, not a file or folder
		return o, nil
	}

	valuesObjects, err := properties.GetValues(id, keysObjects)
	if err != nil {
		return o, fmt.Errorf("unable to get file system values on object '%s': %w", id, err)
	}
	defer valuesObjects.Release()

	o.Name, err = valuesObjects.GetStringValue(WPD_OBJECT_ORIGINAL_FILE_NAME)
	if err != nil {
		return o, fmt.Errorf("unable to get object original file name: %w", err)
	}
	o.Deletable, err = valuesObjects.GetBoolValue(WPD_OBJECT_CAN_DELETE)
	if err != nil {
		return nil, fmt.Errorf("unable to get object can delete: %w", err)
	}
	var pv *PROPVARIANT
	pv, err = valuesObjects.GetValue(WPD_OBJECT_DATE_MODIFIED)
	if err != nil {
		return o, fmt.Errorf("unable to get object modified: %w", err)
	}
	o.Modified = pv.Time()
	var usize uint64
	usize, err = valuesObjects.GetUnsignedLargeIntegerValue(WPD_OBJECT_SIZE)
	if err != nil {
		return o, fmt.Errorf("unable to get object size: %w", err)
	}
	// Do not think overflow is ever an issue here given device memory size
	o.ObjSize = int64(usize)
	return o, nil
}

func (d *Device) fillDeviceInfo() (propSet, error) {
	content, err := d.pdevice.Content()
	if err != nil {
		return nil, fmt.Errorf("failed to get device Content: %w", err)
	}
	defer content.Release()
	properties, err := content.Properties()
	if err != nil {
		return nil, fmt.Errorf("failed to get content Properties: %w", err)
	}
	defer properties.Release()

	type kdef struct {
		name   string
		value  *PropertyKey
		vt     ole.VT
		toType reflect.Type
	}
	kdefs := []kdef{
		{"Firmware", WPD_DEVICE_FIRMWARE_VERSION, ole.VT_LPWSTR, reflect.TypeFor[string]()},
		{"Protocol", WPD_DEVICE_PROTOCOL, ole.VT_LPWSTR, reflect.TypeFor[string]()},
		{"Manufacturer", WPD_DEVICE_MANUFACTURER, ole.VT_LPWSTR, reflect.TypeFor[string]()},
		{"Model", WPD_DEVICE_MODEL, ole.VT_LPWSTR, reflect.TypeFor[string]()},
		{"Serial", WPD_DEVICE_SERIAL_NUMBER, ole.VT_LPWSTR, reflect.TypeFor[string]()},
		{"Name", WPD_DEVICE_FRIENDLY_NAME, ole.VT_LPWSTR, reflect.TypeFor[string]()},
		{"Type", WPD_DEVICE_TYPE, ole.VT_UI4, reflect.TypeFor[WPDDeviceTypes]()},
		{"Transport", WPD_DEVICE_TRANSPORT, ole.VT_UI4, reflect.TypeFor[WPDDeviceTransports]()},
	}

	var keys *IPortableDeviceKeyCollection
	if keys, err = CreatePortableDeviceKeyCollection(); err != nil {
		return nil, err
	}
	defer keys.Release()

	for _, t := range kdefs {
		if err := keys.Add(t.value); err != nil {
			return nil, fmt.Errorf("failed to Add '%s' to keys : %w", t.name, err)
		}
	}

	vals, err := properties.GetValues(WPD_DEVICE_OBJECT_ID, keys)
	if err != nil {
		return nil, fmt.Errorf("failed to GetValues on properties: %w", err)
	}
	defer vals.Release()

	info := make(propSet)
	for _, t := range kdefs {
		switch t.vt {
		case ole.VT_LPWSTR:
			s, err := vals.GetStringValue(t.value)
			if err != nil {
				d.log.Warn("Failed to get string value from properies", zap.String("name", t.name), zap.Error(err))
				break
			}
			info[t.name] = s
		case ole.VT_UI4:
			u, err := vals.GetUnsignedIntegerValue(t.value)
			if err != nil {
				d.log.Warn("Failed to get unsigned integer value from properies", zap.String("name", t.name), zap.Error(err))
				break
			}
			// NOTE: To generalize this we need to check if type is convertible to "toType" and if "toType" implements stringer interface
			// for now we will just assume this is always true
			info[t.name] = reflect.ValueOf(u).Convert(t.toType).MethodByName("String").Call(nil)[0].Interface().(string)
		case ole.VT_VECTOR | ole.VT_UI1:
			buf, err := vals.GetBufferValue(t.value)
			if err != nil {
				d.log.Warn("Failed to get buffer value from properies", zap.String("name", t.name), zap.Error(err))
				break
			}
			info[t.name] = fmt.Sprintf("%x", buf)
		}
	}
	return info, nil
}

func (d *Device) fillStorageInfo() (propSet, error) {
	capabilities, err := d.pdevice.Capabilities()
	if err != nil {
		return nil, fmt.Errorf("failed to get device Capabilites: %w", err)
	}
	defer capabilities.Release()
	categories, err := capabilities.GetFunctionalCategories()
	if err != nil {
		return nil, fmt.Errorf("failed to GetFunctionalCategories from capabilities: %w", err)
	}
	defer categories.Release()
	count, err := categories.GetCount()
	if err != nil {
		return nil, fmt.Errorf("failed to GetCount for functional categories: %w", err)
	}
	found := false
	for i := range count {
		v, err := categories.GetAt(i)
		if err != nil {
			return nil, fmt.Errorf("failed to GetAt(%d) for functional categories: %w", i, err)
		}
		if v.Puuid() != nil && ole.IsEqualGUID(&WPD_FUNCTIONAL_CATEGORY_STORAGE, v.Puuid()) {
			found = true
			break
		}
	}
	if !found {
		return nil, common.ErrNoStorage
	}
	content, err := d.pdevice.Content()
	if err != nil {
		return nil, fmt.Errorf("failed to get device Content: %w", err)
	}
	defer content.Release()
	properties, err := content.Properties()
	if err != nil {
		return nil, fmt.Errorf("failed to get content Properties: %w", err)
	}
	defer properties.Release()

	var keys *IPortableDeviceKeyCollection
	if keys, err = CreatePortableDeviceKeyCollection(); err != nil {
		return nil, err
	}
	defer keys.Release()

	type kdef struct {
		name   string
		value  *PropertyKey
		vt     ole.VT
		toType reflect.Type
		actor  func(objects.ObjectID, any) error
	}
	kdefs := []kdef{
		{"Content Type", WPD_OBJECT_CONTENT_TYPE, ole.VT_CLSID, reflect.TypeFor[ole.GUID](), func(_ objects.ObjectID, v any) error {
			guid, ok := v.(*ole.GUID)
			if !ok || !ole.IsEqualGUID(&WPD_CONTENT_TYPE_FUNCTIONAL_OBJECT, guid) {
				return fmt.Errorf("wrong content type '%s' - not functional", guid)
			}
			return nil
		}},
		{"Functional Category", WPD_FUNCTIONAL_OBJECT_CATEGORY, ole.VT_CLSID, reflect.TypeFor[ole.GUID](), func(oid objects.ObjectID, v any) error {
			guid, ok := v.(*ole.GUID)
			if !ok || !ole.IsEqualGUID(&WPD_FUNCTIONAL_CATEGORY_STORAGE, guid) {
				return fmt.Errorf("wrong object category '%s' - not storage", guid)
			}
			if len(d.storage) > 0 {
				// today all Kindles have single internal storage
				return fmt.Errorf("multiple storage objects found: '%s', '%s'", d.storage, oid)
			}
			d.storage = path.Join(WPD_DEVICE_OBJECT_ID.String(), oid.String())
			return nil
		}},
		{"Display Name", WPD_OBJECT_NAME, ole.VT_LPWSTR, reflect.TypeFor[string](), nil},
		{"Storage Description", WPD_STORAGE_DESCRIPTION, ole.VT_LPWSTR, reflect.TypeFor[string](), nil},
		{"Storage Capacity", WPD_STORAGE_CAPACITY, ole.VT_UI8, reflect.TypeFor[WPDStorageBytes](), func(_ objects.ObjectID, v any) error {
			val, ok := v.(uint64)
			if ok {
				d.availableBytes = int64(val)
			}
			return nil
		}},
		// {"Storage Serial Number", WPD_STORAGE_SERIAL_NUMBER, ole.VT_LPWSTR, reflect.TypeFor[string](), nil},
		{"Storage Free Space", WPD_STORAGE_FREE_SPACE_IN_BYTES, ole.VT_UI8, reflect.TypeFor[WPDStorageBytes](), func(_ objects.ObjectID, v any) error {
			val, ok := v.(uint64)
			if ok {
				d.freeBytes = int64(val)
			}
			return nil
		}},
		{"Storage Access", WPD_STORAGE_ACCESS_CAPABILITY, ole.VT_UI4, reflect.TypeFor[WPDStorageAccessCapability](), func(_ objects.ObjectID, v any) error {
			val, ok := v.(uint32)
			if ok && val == uint32(WPD_STORAGE_ACCESS_CAPABILITY_READWRITE) {
				d.fullAccess = true
			}
			return nil
		}},
		{"File System Type", WPD_STORAGE_FILE_SYSTEM_TYPE, ole.VT_LPWSTR, reflect.TypeFor[string](), nil},
		{"Storage Type", WPD_STORAGE_TYPE, ole.VT_UI4, reflect.TypeFor[WPDStorageType](), nil},
	}
	for _, t := range kdefs {
		if err := keys.Add(t.value); err != nil {
			return nil, fmt.Errorf("failed to Add '%s' to keys : %w", t.name, err)
		}
	}

	info := make(propSet)
	objects, err := content.EnumObjects(0, WPD_DEVICE_OBJECT_ID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to EnumObjects: %w", err)
	}
	defer objects.Release()

	oids, err := objects.Next(1)
	if err != nil {
		return nil, fmt.Errorf("unable to get device storage info: %w", err)
	}

	for _, oid := range oids {
		values, err := properties.GetValues(oid, keys)
		if err != nil {
			return nil, fmt.Errorf("failed to GetValues on properties: %w", err)
		}
		for _, t := range kdefs {
			switch t.vt {
			case ole.VT_CLSID:
				guid, err := values.GetGuidValue(t.value)
				if err != nil {
					return nil, fmt.Errorf("failed to get guid value for '%s' from properies: %w", t.name, err)
				}
				if t.actor != nil {
					if err := t.actor(oid, &guid); err != nil {
						return nil, err
					}
				}
			case ole.VT_LPWSTR:
				str, err := values.GetStringValue(t.value)
				if err != nil {
					d.log.Warn("Failed to get string value from properies", zap.String("name", t.name), zap.Error(err))
					break
				}
				info[t.name] = str
				if t.actor != nil {
					if err := t.actor(oid, str); err != nil {
						return nil, err
					}
				}
			case ole.VT_UI4:
				u, err := values.GetUnsignedIntegerValue(t.value)
				if err != nil {
					d.log.Warn("Failed to get unsigned integer value from properies", zap.String("name", t.name), zap.Error(err))
					break
				}
				info[t.name] = reflect.ValueOf(u).Convert(t.toType).MethodByName("String").Call(nil)[0].Interface().(string)
				if t.actor != nil {
					if err := t.actor(oid, u); err != nil {
						return nil, err
					}
				}
			case ole.VT_UI8:
				u, err := values.GetUnsignedLargeIntegerValue(t.value)
				if err != nil {
					d.log.Warn("Failed to get unsigned large integer value from properies", zap.String("name", t.name), zap.Error(err))
					break
				}
				info[t.name] = reflect.ValueOf(u).Convert(t.toType).MethodByName("String").Call(nil)[0].Interface().(string)
				if t.actor != nil {
					if err := t.actor(oid, u); err != nil {
						return nil, err
					}
				}
			}
		}
		values.Release()
	}
	return info, nil
}

func createObjectValues(parent objects.ObjectID, name string, objType *ole.GUID, size int64) (*IPortableDeviceValues, error) {
	values, err := CreatePortableDeviceValues()
	if err != nil {
		return nil, err
	}

	if err := values.SetUTF16Value(WPD_OBJECT_PARENT_ID, parent); err != nil {
		return nil, fmt.Errorf("failed to set WPD_OBJECT_PARENT_ID: %w", err)
	}
	if err := values.SetStringValue(WPD_OBJECT_NAME, name); err != nil {
		return nil, fmt.Errorf("failed to set WPD_OBJECT_NAME: %w", err)
	}
	if err := values.SetStringValue(WPD_OBJECT_ORIGINAL_FILE_NAME, name); err != nil {
		return nil, fmt.Errorf("failed to set WPD_OBJECT_ORIGINAL_FILE_NAME: %w", err)
	}
	if err := values.SetGuidValue(WPD_OBJECT_FORMAT, &WPD_OBJECT_FORMAT_UNSPECIFIED); err != nil {
		return nil, fmt.Errorf("failed to set WPD_OBJECT_FORMAT: %w", err)
	}
	if err := values.SetGuidValue(WPD_OBJECT_CONTENT_TYPE, objType); err != nil {
		return nil, fmt.Errorf("failed to set WPD_OBJECT_CONTENT_TYPE: %w", err)
	}
	ts := time.Now()
	vd, err := NewPropVariantFromTime(ts)
	if err != nil {
		return nil, fmt.Errorf("failed to create PROPVARIANT from time '%s': %w", ts, err)
	}
	defer vd.Clear()
	if err := values.SetValue(WPD_OBJECT_DATE_CREATED, vd); err != nil {
		return nil, fmt.Errorf("failed to set WPD_OBJECT_DATE_CREATED: %w", err)
	}
	if err := values.SetValue(WPD_OBJECT_DATE_MODIFIED, vd); err != nil {
		return nil, fmt.Errorf("failed to set WPD_OBJECT_DATE_MODIFIED: %w", err)
	}
	if !ole.IsEqualGUID(&WPD_CONTENT_TYPE_FOLDER, objType) {
		if err := values.SetUnsignedLargeIntegerValue(WPD_OBJECT_SIZE, uint64(size)); err != nil {
			return nil, fmt.Errorf("failed to set WPD_OBJECT_SIZE: %w", err)
		}
	}
	return values, nil
}

// If serial number is given in configuration - attempts to find this device, otherwise assumes
// that there could only be a single Kindle connected at a time, picks
// first one found and returns an error if there are no supported devices.
// NOTE: outputs debug data for all enumerated devices
func pickDevice(pdm *IPortableDeviceManager, serial string, log *zap.Logger) (common.PnPDeviceID, error) {
	if err := pdm.RefreshDeviceList(); err != nil {
		return nil, err
	}
	ids, err := pdm.GetDevices()
	if err != nil {
		return nil, fmt.Errorf("unable to GetDevices: %w", err)
	}
	var result common.PnPDeviceID
	log.Debug("MTP Devices", zap.Int("count", len(ids)))
	for _, id := range ids {
		name, err := pdm.GetDeviceFriendlyName(id)
		if err != nil {
			return nil, fmt.Errorf("unable to GetDeviceFriendlyName for '%s': %w", id, err)
		}
		descr, err := pdm.GetDeviceDescription(id)
		if err != nil {
			return nil, fmt.Errorf("unable to GetDeviceDescription for '%s': %w", id, err)
		}
		mfr, err := pdm.GetDeviceManufacturer(id)
		if err != nil {
			return nil, fmt.Errorf("unable to GetDeviceManufacturer for '%s': %w", id, err)
		}
		vid, pid := id.VendorID(), id.ProductID()
		supported := common.IsKindleDevice(common.ProtocolMTP, vid, pid)
		log.Debug("Driver Info",
			zap.Stringer("PnP ID", id),
			zap.Bool("supported", supported),
		)

		if !supported {
			continue
		}

		if len(serial) > 0 {
			if !strings.EqualFold(serial, id.Serial()) {
				continue
			}
			// we are targeting a specific device
		} else {
			if len(result) != 0 {
				continue
			}
			// pick the first supported device
		}
		result = id

		log.Debug("Device Info",
			zap.Stringer("Device ID", id),
			zap.String("Name", name),
			zap.String("Description", descr),
			zap.String("Manufacturer", mfr),
			zap.String("VendorID", fmt.Sprintf("0x%04X", vid)),
			zap.String("ProductID", fmt.Sprintf("0x%04X", pid)),
			zap.String("Serial", id.Serial()),
			zap.Bool("supported", supported),
		)
	}
	if len(result) == 0 {
		return nil, common.ErrNoDevice
	}
	return result, nil
}

func prepareClientInfo() (ci *IPortableDeviceValues, err error) {
	defer func() {
		if ci != nil && err != nil {
			ci.Release()
			ci = nil
		}
	}()

	ci, err = CreatePortableDeviceValues()
	if err != nil {
		return
	}

	err = ci.SetStringValue(WPD_CLIENT_NAME, misc.GetAppName())
	if err != nil {
		err = fmt.Errorf("failed to set WPD_CLIENT_NAME: %w", err)
		return
	}

	var major, minor, revision uint32
	matches := regexp.MustCompile(`([0-9]+)\.([0-9]+)\.?([0-9]*)[-a-zA-Z0-9+]*`).FindStringSubmatch(misc.GetVersion())
	if len(matches) == 4 {
		if id, err := strconv.ParseInt(matches[1], 10, 32); err == nil {
			major = uint32(id)
		}
		if id, err := strconv.ParseInt(matches[2], 10, 32); err == nil {
			minor = uint32(id)
		}
		if id, err := strconv.ParseInt(matches[3], 10, 32); err == nil {
			revision = uint32(id)
		}
	}
	err = ci.SetUnsignedIntegerValue(WPD_CLIENT_MAJOR_VERSION, major)
	if err != nil {
		err = fmt.Errorf("failed to set WPD_CLIENT_MAJOR_VERSION: %w", err)
		return
	}
	err = ci.SetUnsignedIntegerValue(WPD_CLIENT_MINOR_VERSION, minor)
	if err != nil {
		err = fmt.Errorf("failed to set WPD_CLIENT_MINOR_VERSION: %w", err)
		return
	}
	err = ci.SetUnsignedIntegerValue(WPD_CLIENT_REVISION, revision)
	if err != nil {
		err = fmt.Errorf("failed to set WPD_CLIENT_REVISION: %w", err)
		return
	}
	err = ci.SetUnsignedIntegerValue(WPD_CLIENT_SECURITY_QUALITY_OF_SERVICE, windows.SECURITY_IMPERSONATION)
	if err != nil {
		err = fmt.Errorf("failed to set WPD_CLIENT_SECURITY_QUALITY_OF_SERVICE: %w", err)
		return
	}
	return
}

func getFullPathAt(infos []*objects.ObjectInfo, at int) string {
	if len(infos) == 0 || at < 0 || at >= len(infos) {
		return ""
	}

	fullPath := infos[at].Name
	for {
		found := slices.IndexFunc(infos, func(o *objects.ObjectInfo) bool {
			return slices.Equal(o.Oid, infos[at].OidParent)
		})
		if found == -1 {
			break
		}
		fullPath, at = path.Join(infos[found].Name, fullPath), found
	}
	return fullPath
}

func init() {
	// initialize WPD_DEVICE_OBJECT_ID for global usage
	var err error
	WPD_DEVICE_OBJECT_ID, err = windows.UTF16FromString("DEVICE")
	if err != nil {
		panic(err)
	}
}
