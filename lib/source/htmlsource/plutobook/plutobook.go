package plutobook

/*
#cgo pkg-config: plutobook
#include <plutobook/plutobook.h>

typedef const char* ccharptr_t;
extern plutobook_resource_data_t* goFetcherCallback(void* closure, ccharptr_t url);

__attribute__((weak))
plutobook_resource_data_t* fetchCallback(void* closure, const char* url) {
	return goFetcherCallback(closure, url);
}

*/
import "C"
import (
	"fmt"
	"unsafe"

	gopointer "github.com/mattn/go-pointer"
)

const Pixels = C.PLUTOBOOK_UNITS_PX

type Plutobook struct {
	book          *C.plutobook_t
	fetchCallback func(url string) *Resource
}

type Resource struct {
	resource *C.plutobook_resource_data_t
}

type MediaType C.plutobook_media_type_t

const (
	MediaTypePrint  MediaType = C.PLUTOBOOK_MEDIA_TYPE_PRINT
	MediaTypeScreen MediaType = C.PLUTOBOOK_MEDIA_TYPE_SCREEN
)

type ImageFormat C.plutobook_image_format_t

const (
	ImageFormatInvalid ImageFormat = C.PLUTOBOOK_IMAGE_FORMAT_INVALID
	ImageFormatARGB32              = C.PLUTOBOOK_IMAGE_FORMAT_ARGB32
	ImageFormatRGB24               = C.PLUTOBOOK_IMAGE_FORMAT_RGB24
	ImageFormatA8                  = C.PLUTOBOOK_IMAGE_FORMAT_A8
	ImageFormatA1                  = C.PLUTOBOOK_IMAGE_FORMAT_A1
)

func Version() int {
	return int(C.plutobook_version())
}

func VersionString() string {
	return C.GoString(C.plutobook_version_string())
}

func BuildInfo() string {
	return C.GoString(C.plutobook_build_info())
}

func GetErrorMessage() string {
	return C.GoString(C.plutobook_get_error_message())
}

type PageSize struct {
	Width  float32
	Height float32
}

type Margins struct {
	Top    float32
	Right  float32
	Bottom float32
	Left   float32
}

func New(pageSize *PageSize, margins *Margins, mediaType MediaType) *Plutobook {
	result := &Plutobook{}
	cPageSize := C.plutobook_page_size_t{width: C.float(pageSize.Width), height: C.float(pageSize.Height)}

	cMargins := C.plutobook_page_margins_t{}
	cMargins.top = C.float(margins.Top)
	cMargins.right = C.float(margins.Right)
	cMargins.bottom = C.float(margins.Bottom)
	cMargins.left = C.float(margins.Left)

	result.book = C.plutobook_create(cPageSize, cMargins, C.plutobook_media_type_t(mediaType))
	return result
}

func (p *Plutobook) LoadUrl(url, userStyle, userScript string) error {
	curl := C.CString(url)
	cstyle := C.CString(userStyle)
	cscript := C.CString(userScript)
	if C.plutobook_load_url(p.book, curl, cstyle, cscript) {
		return nil
	} else {
		return fmt.Errorf("Failed to load %s: %s", url, GetErrorMessage())
	}
}

func (p *Plutobook) LoadHtml(html, userStyle, userScript, baseUrl string) error {
	chtml := C.CString(html)
	cstyle := C.CString(userStyle)
	cscript := C.CString(userScript)
	cbaseUrl := C.CString(baseUrl)
	if C.plutobook_load_html(p.book, chtml, -1, cstyle, cscript, cbaseUrl) {
		return nil
	} else {
		return fmt.Errorf("Failed to load: %s", GetErrorMessage())
	}
}

func (p *Plutobook) WritePNG(filename string, width, height int) error {
	cfilename := C.CString(filename)
	if C.plutobook_write_to_png(p.book, cfilename, C.int(width), C.int(height)) {
		return nil
	} else {
		return fmt.Errorf("Failed to write %s: %s", filename, GetErrorMessage())
	}
}

func (p *Plutobook) RenderPage(canvas *Canvas, page int) {
	C.plutobook_render_page(p.book, canvas.canvas, C.uint(page))
}

func (p *Plutobook) RenderDocument(canvas *Canvas) {
	C.plutobook_render_document(p.book, canvas.canvas)
}

func (p *Plutobook) RenderDocumentRect(canvas *Canvas, x, y, width, height float32) {
	C.plutobook_render_document_rect(p.book, canvas.canvas, C.float(x), C.float(y), C.float(width), C.float(height))
}

func FetchUrl(url string) *Resource {
	curl := C.CString(url)
	return &Resource{resource: C.plutobook_fetch_url(curl)}
}

type ResourceCallback func(url string) *Resource

func (p *Plutobook) SetCustomResourceFetcher(callback func(url string) *Resource) {
	p.fetchCallback = callback
	ptr := gopointer.Save(p)
	//defer gopointer.Unref(ptr)

	C.plutobook_set_custom_resource_fetcher(p.book, C.plutobook_resource_fetch_callback_t(C.fetchCallback), ptr)
}

//export goFetcherCallback
func goFetcherCallback(closure unsafe.Pointer, url C.ccharptr_t) *C.plutobook_resource_data_t {
	realUrl := C.GoString(url)
	book := gopointer.Restore(closure).(*Plutobook)
	result := book.fetchCallback(realUrl)
	return result.resource
}

type Canvas struct {
	canvas *C.plutobook_canvas_t
	width  int
	height int
	format ImageFormat
}

func NewCanvas(width, height int, format ImageFormat) *Canvas {
	result := &Canvas{width: width, height: height, format: format}
	result.canvas = C.plutobook_image_canvas_create(C.int(width), C.int(height), C.plutobook_image_format_t(format))
	return result
}

func (c *Canvas) GetData() []byte {
	size := c.width * c.height
	switch c.format {
	case ImageFormatARGB32:
		size *= 4
	case ImageFormatRGB24:
		size *= 3
	case ImageFormatA1:
		size /= 8
	}
	var data *C.uchar = C.plutobook_image_canvas_get_data(c.canvas)
	//return unsafe.Slice(data, size)
	return C.GoBytes(unsafe.Pointer(data), C.int(size))
}

func (c *Canvas) GetWidth() int {
	return c.width
}

func (c *Canvas) GetHeight() int {
	return c.height
}

func (c *Canvas) GetFormat() ImageFormat {
	return c.format
}
