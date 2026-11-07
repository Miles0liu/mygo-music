//go:build windows

package app

import (
	"encoding/binary"
	"log"
	"syscall"
	"unsafe"

	"github.com/egoist/mygo"
	"golang.org/x/sys/windows"
)

// ITaskbarList3 and a window subclass live on MyGo's UI thread. Explorer
// recreates the toolbar after TaskbarButtonCreated, including after a restart.
type desktopShell struct {
	hwnd     uintptr
	taskbar  *taskbarObject
	icons    [4]uintptr
	added    bool
	title    string
	playing  bool
	hasTrack bool
}

type thumbButton struct {
	Mask, ID, Bitmap uint32
	Icon             uintptr
	Tip              [260]uint16
	Flags            uint32
}

var (
	shellUser      = windows.NewLazySystemDLL("user32.dll")
	shellControls  = windows.NewLazySystemDLL("comctl32.dll")
	shellOLE       = windows.NewLazySystemDLL("ole32.dll")
	setSubclass    = shellControls.NewProc("SetWindowSubclass")
	removeSubclass = shellControls.NewProc("RemoveWindowSubclass")
	defSubclass    = shellControls.NewProc("DefSubclassProc")
	shellApps      = map[uintptr]*app{}
	shellCallback  uintptr
	taskbarCreated uintptr
)

func init() { shellCallback = syscall.NewCallback(taskbarWindowProc) }

type taskbarObject struct{ table *[21]uintptr }

func taskbarCall(object *taskbarObject, method int, args ...uintptr) uintptr {
	result, _, _ := syscall.SyscallN(object.table[method], append([]uintptr{uintptr(unsafe.Pointer(object))}, args...)...)
	return result
}

func (a *app) initDesktop() {
	s := &a.shell
	s.hwnd = a.win.NativeHandle()
	shellApps[s.hwnd] = a
	name, _ := windows.UTF16PtrFromString("TaskbarButtonCreated")
	taskbarCreated, _, _ = shellUser.NewProc("RegisterWindowMessageW").Call(uintptr(unsafe.Pointer(name)))
	ok, _, err := setSubclass.Call(s.hwnd, shellCallback, 1, 0)
	if ok == 0 {
		log.Printf("taskbar subclass: %v", err)
		delete(shellApps, s.hwnd)
		return
	}
	class, _ := windows.GUIDFromString("{56FDF344-FD6D-11D0-958A-006097C9A090}")
	iid, _ := windows.GUIDFromString("{EA1AFB91-9E28-4B86-90E9-9E9F8A5EEFAF}")
	hr, _, _ := shellOLE.NewProc("CoCreateInstance").Call(uintptr(unsafe.Pointer(&class)), 0, 1,
		uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&s.taskbar)))
	if int32(hr) < 0 || s.taskbar == nil {
		log.Printf("taskbar unavailable: 0x%x", hr)
		return
	}
	if hr = taskbarCall(s.taskbar, 3); int32(hr) < 0 {
		taskbarCall(s.taskbar, 2)
		s.taskbar = nil
		return
	}
	for i := range s.icons {
		s.icons[i] = transportIcon(i)
	}
	a.syncDesktop()
}

func taskbarWindowProc(hwnd, msg, wp, lp, id, data uintptr) uintptr {
	if a := shellApps[hwnd]; a != nil {
		switch {
		case msg == taskbarCreated && taskbarCreated != 0:
			a.shell.added = false
			a.syncDesktop()
		case msg == 0x111 && wp>>16 == 0x1800:
			command := wp & 0xffff
			if command >= 4100 && command <= 4102 {
				a.win.Update(func() {
					switch command {
					case 4100:
						a.pl.Prev()
					case 4101:
						a.pl.TogglePlay()
					case 4102:
						a.pl.Next()
					}
					a.syncDesktop()
				})
				return 0
			}
		case msg == 0x82: // WM_NCDESTROY
			a.closeDesktop()
		}
	}
	result, _, _ := defSubclass.Call(hwnd, msg, wp, lp)
	return result
}

func (a *app) syncDesktop() {
	if a.win == nil {
		return
	}
	s := &a.shell
	title := mygo.App.Name()
	tr := a.pl.Current()
	if tr != nil {
		title = tr.Name
		if artists := tr.ArtistNames(); artists != "" {
			title += " — " + artists
		}
		if !a.pl.Playing() {
			title += " · " + T("Paused")
		}
		title += " | " + mygo.App.Name()
	}
	changed := title != s.title || s.playing != a.pl.Playing() || s.hasTrack != (tr != nil)
	s.playing, s.hasTrack = a.pl.Playing(), tr != nil
	if title != s.title {
		a.win.SetTitle(title)
		s.title = title
	}
	if s.taskbar == nil || (!changed && s.added) {
		return
	}
	tip, _ := windows.UTF16PtrFromString(title)
	taskbarCall(s.taskbar, 19, s.hwnd, uintptr(unsafe.Pointer(tip)))
	middle, middleName := 1, T("Play")
	if a.pl.Playing() {
		middle, middleName = 2, T("Pause")
	}
	buttons := [3]thumbButton{}
	names := []string{T("Previous"), middleName, T("Next")}
	icons := []uintptr{s.icons[0], s.icons[middle], s.icons[3]}
	for i := range buttons {
		buttons[i] = thumbButton{Mask: 2 | 4 | 8, ID: uint32(4100 + i), Icon: icons[i]}
		if tr == nil {
			buttons[i].Flags = 1
		}
		text, _ := windows.UTF16FromString(names[i])
		copy(buttons[i].Tip[:259], text)
	}
	method := 15
	if s.added {
		method = 16
	}
	hr := taskbarCall(s.taskbar, method, s.hwnd, 3, uintptr(unsafe.Pointer(&buttons[0])))
	if int32(hr) >= 0 {
		s.added = true
	}
}

func (a *app) closeDesktop() {
	s := &a.shell
	if s.hwnd != 0 {
		removeSubclass.Call(s.hwnd, shellCallback, 1)
		delete(shellApps, s.hwnd)
	}
	if s.taskbar != nil {
		taskbarCall(s.taskbar, 2)
	}
	for _, icon := range s.icons {
		if icon != 0 {
			shellUser.NewProc("DestroyIcon").Call(icon)
		}
	}
	*s = desktopShell{}
}

// A 32-bit icon resource, rasterized at the shell's icon size. The shell
// copies these handles for its toolbar and owns the disabled/hover visuals.
func transportIcon(kind int) uintptr {
	size, _, _ := shellUser.NewProc("GetSystemMetrics").Call(11) // SM_CXICON
	if size < 16 || size > 256 {
		size = 32
	}
	n := int(size)
	maskStride := ((n + 31) / 32) * 4
	data := make([]byte, 40+n*n*4+maskStride*n)
	binary.LittleEndian.PutUint32(data[0:], 40)
	binary.LittleEndian.PutUint32(data[4:], uint32(n))
	binary.LittleEndian.PutUint32(data[8:], uint32(n*2))
	binary.LittleEndian.PutUint16(data[12:], 1)
	binary.LittleEndian.PutUint16(data[14:], 32)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			px, py := float64(x)/float64(n), float64(y)/float64(n)
			if kind == 0 {
				px = 1 - px
			}
			on := false
			if kind == 2 {
				on = py > 0.22 && py < 0.78 && ((px > 0.28 && px < 0.43) || (px > 0.57 && px < 0.72))
			} else {
				d := py - 0.5
				if d < 0 {
					d = -d
				}
				on = px > 0.28 && px < 0.76 && d < (0.76-px)*0.65
				if kind != 1 {
					on = on || (px > 0.77 && px < 0.84 && py > 0.22 && py < 0.78)
				}
			}
			offset := 40 + ((n-1-y)*n+x)*4
			if on {
				copy(data[offset:offset+4], []byte{245, 245, 245, 255})
			}
		}
	}
	icon, _, _ := shellUser.NewProc("CreateIconFromResourceEx").Call(uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), 1, 0x30000, size, size, 0)
	return icon
}
