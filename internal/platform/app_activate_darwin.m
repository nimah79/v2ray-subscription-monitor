#import <AppKit/AppKit.h>
#import <Carbon/Carbon.h>
#import <Foundation/Foundation.h>

extern void platformAppDidBecomeActiveGo(void);

static BOOL gAppleEventRegistered = NO;

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
