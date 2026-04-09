#import <AppKit/AppKit.h>
#import <Carbon/Carbon.h>
#import <Foundation/Foundation.h>

extern void platformAppDidBecomeActiveGo(void);
extern void platformQuitAppleEventGo(void);

static BOOL gAppleEventRegistered = NO;
static BOOL gQuitAppleEventRegistered = NO;

// kAEReopenApplication = Dock icon click for a running app. Do NOT use
// NSApplicationDidBecomeActiveNotification here — dismissing an NSAlert also becomes active and would
// incorrectly re-open the main window while tray-only.

@interface V2RAYDockReopenHandler : NSObject
@end

@implementation V2RAYDockReopenHandler
- (void)handleReopenAppleEvent:(NSAppleEventDescriptor *)event withReplyEvent:(NSAppleEventDescriptor *)replyEvent {
	(void)event;
	(void)replyEvent;
	platformAppDidBecomeActiveGo();
}
@end

static V2RAYDockReopenHandler *gReopenHandler = nil;

void platform_register_did_become_active(void) {
	if (gAppleEventRegistered) {
		return;
	}
	gAppleEventRegistered = YES;
	if (gReopenHandler == nil) {
		gReopenHandler = [V2RAYDockReopenHandler new];
	}
	[[NSAppleEventManager sharedAppleEventManager] setEventHandler:gReopenHandler
	                                                   andSelector:@selector(handleReopenAppleEvent:withReplyEvent:)
	                                                 forEventClass:kCoreEventClass
	                                                    andEventID:kAEReopenApplication];
}

@interface V2RAYQuitAppleEventHandler : NSObject
@end

@implementation V2RAYQuitAppleEventHandler
- (void)handleQuitAppleEvent:(NSAppleEventDescriptor *)event withReplyEvent:(NSAppleEventDescriptor *)replyEvent {
	(void)event;
	(void)replyEvent;
	platformQuitAppleEventGo();
}
@end

static V2RAYQuitAppleEventHandler *gQuitHandler = nil;

// Dock “Quit” and Cmd+Q deliver kAEQuitApplication; Fyne’s systray external loop does not set an
// NSApplicationDelegate that forwards this to the GLFW run loop.
void platform_register_quit_apple_event(void) {
	if (gQuitAppleEventRegistered) {
		return;
	}
	gQuitAppleEventRegistered = YES;
	if (gQuitHandler == nil) {
		gQuitHandler = [V2RAYQuitAppleEventHandler new];
	}
	[[NSAppleEventManager sharedAppleEventManager] setEventHandler:gQuitHandler
	                                                   andSelector:@selector(handleQuitAppleEvent:withReplyEvent:)
	                                                 forEventClass:kCoreEventClass
	                                                    andEventID:kAEQuitApplication];
}
