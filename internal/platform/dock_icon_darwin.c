#include <pthread.h>
#include <dispatch/dispatch.h>
#include <objc/message.h>
#include <objc/runtime.h>

static id shared_ns_application(void) {
	Class cls = objc_getClass("NSApplication");
	if (cls == NULL) {
		return NULL;
	}
	SEL sel = sel_registerName("sharedApplication");
	id (*msgSend)(Class, SEL) = (id (*)(Class, SEL))objc_msgSend;
	return msgSend(cls, sel);
}

static void ensure_ns_application_on_main(void) {
	id app = shared_ns_application();
	if (app == NULL) {
		return;
	}
	SEL sel = sel_registerName("setActivationPolicy:");
	BOOL (*setPol)(id, SEL, long) = (BOOL (*)(id, SEL, long))objc_msgSend;
	(void)setPol(app, sel, 0); /* NSApplicationActivationPolicyRegular */
}

// Creates NSApplication and requests a regular Dock-capable activation policy as early as possible.
void platform_ensure_ns_application(void) {
	if (pthread_main_np() != 0) {
		ensure_ns_application_on_main();
	} else {
		dispatch_sync(dispatch_get_main_queue(), ^{
			ensure_ns_application_on_main();
		});
	}
}

static void set_dock_icon_from_png_on_main(const void *bytes, size_t len) {
	if (bytes == NULL || len == 0) {
		return;
	}

	ensure_ns_application_on_main();

	Class nsDataClass = objc_getClass("NSData");
	if (nsDataClass == NULL) {
		return;
	}
	SEL dataSel = sel_registerName("dataWithBytes:length:");
	id (*dataWithBytes)(Class, SEL, const void *, unsigned long) =
		(id (*)(Class, SEL, const void *, unsigned long))objc_msgSend;
	id data = dataWithBytes(nsDataClass, dataSel, bytes, (unsigned long)len);
	if (data == NULL) {
		return;
	}

	Class nsImageClass = objc_getClass("NSImage");
	if (nsImageClass == NULL) {
		return;
	}
	id (*allocImg)(Class, SEL) = (id (*)(Class, SEL))objc_msgSend;
	id imgAlloc = allocImg(nsImageClass, sel_registerName("alloc"));
	if (imgAlloc == NULL) {
		return;
	}
	SEL initSel = sel_registerName("initWithData:");
	id (*initImg)(id, SEL, id) = (id (*)(id, SEL, id))objc_msgSend;
	id img = initImg(imgAlloc, initSel, data);
	if (img == NULL) {
		SEL releaseSel = sel_registerName("release");
		((void (*)(id, SEL))objc_msgSend)(imgAlloc, releaseSel);
		return;
	}

	id app = shared_ns_application();
	if (app != NULL) {
		SEL setIconSel = sel_registerName("setApplicationIconImage:");
		void (*setIcon)(id, SEL, id) = (void (*)(id, SEL, id))objc_msgSend;
		setIcon(app, setIconSel, img);
	}

	SEL releaseSel = sel_registerName("release");
	((void (*)(id, SEL))objc_msgSend)(img, releaseSel);
}

// Runs on AppKit main thread; uses PNG bytes only during the call (caller must keep alive).
void platform_set_dock_icon_from_png(const void *bytes, size_t len) {
	if (pthread_main_np() != 0) {
		set_dock_icon_from_png_on_main(bytes, len);
	} else {
		dispatch_sync(dispatch_get_main_queue(), ^{
			set_dock_icon_from_png_on_main(bytes, len);
		});
	}
}
