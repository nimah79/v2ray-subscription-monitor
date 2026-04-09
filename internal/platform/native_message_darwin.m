#import <AppKit/AppKit.h>
#import <CoreFoundation/CoreFoundation.h>
#import <dispatch/dispatch.h>
#import <pthread.h>

extern void goAfterNativeModalDialog(void);

static void show_native_alert(NSString *title, NSString *msg) {
	NSApplication *app = [NSApplication sharedApplication];
	// Accessory (tray-only) policy suppresses foreground alerts; become a regular app for the modal.
	[app setActivationPolicy:NSApplicationActivationPolicyRegular];
	[app unhide:nil];
	[app activateIgnoringOtherApps:YES];

	NSAlert *alert = [[NSAlert alloc] init];
	[alert setMessageText:title];
	[alert setInformativeText:msg];
	[alert setAlertStyle:NSAlertStyleWarning];
	[alert addButtonWithTitle:@"OK"];

	// NSRunLoop runMode loops here were removed: pumping the AppKit loop while Fyne/GLFW owns the
	// main thread caused re-entrancy crashes (e.g. with beginSheetModalForWindow).
	[alert layout];
	NSWindow *aw = [alert window];
	if (aw != nil) {
		[aw setLevel:NSModalPanelWindowLevel];
		[aw center];
		[aw orderFrontRegardless];
	}
	[app activateIgnoringOtherApps:YES];

	[alert runModal];

	goAfterNativeModalDialog();
}

void platform_show_native_info(const char *title, const char *message) {
	if (title == NULL || message == NULL) {
		return;
	}
	NSString *t = [NSString stringWithUTF8String:title];
	NSString *m = [NSString stringWithUTF8String:message];
	if (t == nil) {
		t = @"";
	}
	if (m == nil) {
		m = @"";
	}
	if (pthread_main_np() != 0) {
		show_native_alert(t, m);
	} else {
		dispatch_async(dispatch_get_main_queue(), ^{
			CFRunLoopWakeUp(CFRunLoopGetMain());
			show_native_alert(t, m);
		});
	}
}
