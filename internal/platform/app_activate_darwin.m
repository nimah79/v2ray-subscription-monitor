#import <AppKit/AppKit.h>
#import <Carbon/Carbon.h>
#import <Foundation/Foundation.h>

extern void platformAppDidBecomeActiveGo(void);

static id becomeActiveObserver = nil;
static BOOL gAppleEventRegistered = NO;

// Receives kAEReopenApplication (Dock icon click) without touching NSApp.delegate — safe alongside GLFW/Fyne.

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
	if (becomeActiveObserver == nil) {
		NSNotificationCenter *nc = [NSNotificationCenter defaultCenter];
		becomeActiveObserver = [nc addObserverForName:NSApplicationDidBecomeActiveNotification
		                                         object:nil
		                                          queue:[NSOperationQueue mainQueue]
		                                     usingBlock:^(__unused NSNotification *note) {
			                                     platformAppDidBecomeActiveGo();
		                                     }];
	}

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
