#include <dispatch/dispatch.h>
#include <pthread.h>
#include <objc/message.h>
#include <objc/runtime.h>

// NSApplicationActivationPolicyRegular = 0, Accessory = 1

static id shared_ns_application(void) {
	Class cls = objc_getClass("NSApplication");
	if (cls == NULL) {
		return NULL;
	}
	SEL sel = sel_registerName("sharedApplication");
	id (*msgSend)(Class, SEL) = (id (*)(Class, SEL))objc_msgSend;
	return msgSend(cls, sel);
}

static void apply_policy_on_app(id app, long policy) {
	if (app == NULL) {
		return;
	}
	SEL sel = sel_registerName("setActivationPolicy:");
	BOOL (*msgSend)(id, SEL, long) = (BOOL (*)(id, SEL, long))objc_msgSend;
	msgSend(app, sel, policy);
}

void platform_apply_activation_policy(long policy) {
	dispatch_async(dispatch_get_main_queue(), ^{
		apply_policy_on_app(shared_ns_application(), policy);
	});
}

void platform_sync_activation_policy(long policy) {
	if (pthread_main_np() != 0) {
		apply_policy_on_app(shared_ns_application(), policy);
	} else {
		dispatch_sync(dispatch_get_main_queue(), ^{
			apply_policy_on_app(shared_ns_application(), policy);
		});
	}
}
