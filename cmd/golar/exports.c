#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>

#ifdef GOLAR_NAPI_DYNAMIC
#include <dlfcn.h>
#include <pthread.h>
#endif

#define NAPI_VERSION 6
#include "../../napi-include/node_api.h"

#include "exports.h"

#ifdef GOLAR_NAPI_DYNAMIC
#define X(NAME) static __typeof__(&NAME) golar_##NAME##_ptr;
#include "napi_dynamic_symbols.inc"
#undef X

static int golar_resolve_symbol(void **target, const char *name) {
	dlerror();
	*target = dlsym(RTLD_DEFAULT, name);
	const char *err = dlerror();
	if (err != NULL) {
		fprintf(stderr, "golar.node: failed to resolve %s: %s\n", name, err);
		return 0;
	}
	return 1;
}

static bool golar_init_napi_symbols_once_ok = false;
static void golar_init_napi_symbols_once() {
#define X(NAME) \
	if (!golar_resolve_symbol((void **)&golar_##NAME##_ptr, #NAME)) { \
		return; \
	}
#include "napi_dynamic_symbols.inc"
#undef X
	golar_init_napi_symbols_once_ok = true;
}

static pthread_once_t golar_napi_symbols_once = PTHREAD_ONCE_INIT;
static bool golar_init_napi_symbols(void) {
	if (pthread_once(&golar_napi_symbols_once, golar_init_napi_symbols_once) != 0) {
		fprintf(stderr, "golar.node: pthread_once failed\n");
		return false;
	}

	return golar_init_napi_symbols_once_ok;
}

#include "napi_dynamic_aliases.h"
#endif

#define METHOD_DESCRIPTOR_STR(NAME, VALUE) (napi_property_descriptor){ \
	.utf8name = NAME, \
	.name = NULL, \
	.method = VALUE, \
	.getter = NULL, \
	.setter = NULL, \
	.value = NULL, \
	.attributes = napi_enumerable, \
	.data = NULL \
}

static napi_value setSyncBuffer_bridge(napi_env env, napi_callback_info info) {
	size_t argc = 2;
	napi_value argv[argc];
	napi_get_cb_info(env, info, &argc, argv, NULL, NULL);

	uint32_t thread_id;
	napi_get_value_uint32(env, argv[0], &thread_id);

	size_t len = 0;
	void *sync_buffer;
	napi_get_arraybuffer_info(env, argv[1], &sync_buffer, &len);
	golar_js_setSyncBuffer(thread_id, (uintptr_t)sync_buffer, len);

	return NULL;
}

static napi_value linter_RuleTesterLint_bridge(napi_env env, napi_callback_info info) {
	size_t argc = 4;
	napi_value argv[4];
	napi_get_cb_info(env, info, &argc, argv, NULL, NULL);

	char *files_data;
	size_t files_len;
	napi_get_value_string_utf8(env, argv[0], NULL, 0, &files_len);
	files_len++;
	files_data = malloc(sizeof(char) * files_len);
	napi_get_value_string_utf8(env, argv[0], files_data, files_len, &files_len);

	char *fileName_data;
	size_t fileName_len;
	napi_get_value_string_utf8(env, argv[1], NULL, 0, &fileName_len);
	fileName_len++;
	fileName_data = malloc(sizeof(char) * fileName_len);
	napi_get_value_string_utf8(env, argv[1], fileName_data, fileName_len, &fileName_len);

	char *ruleName_data;
	size_t ruleName_len;
	napi_get_value_string_utf8(env, argv[2], NULL, 0, &ruleName_len);
	ruleName_len++;
	ruleName_data = malloc(sizeof(char) * ruleName_len);
	napi_get_value_string_utf8(env, argv[2], ruleName_data, ruleName_len, &ruleName_len);

	char *options_data;
	size_t options_len;
	napi_get_value_string_utf8(env, argv[3], NULL, 0, &options_len);
	options_len++;
	options_data = malloc(sizeof(char) * options_len);
	napi_get_value_string_utf8(env, argv[3], options_data, options_len, &options_len);

	struct golar_js_linter_RuleTesterLint_return res = golar_js_linter_RuleTesterLint(files_data, files_len, fileName_data, fileName_len, ruleName_data, ruleName_len, options_data, options_len);

	free(files_data);
	free(fileName_data);
	free(ruleName_data);
	free(options_data);

	napi_value res_str;
	napi_create_string_utf8(env, res.r0, res.r1, &res_str);
	free(res.r0);

	return res_str;
}

// The thread-safe functions' call_js callbacks. N-API calls them with a NULL
// env and js_callback while it tears a function down (its thread is exiting):
// JS can no longer run then, so each returns at once.

// Calls a done callback with no arguments. An exception the callback throws
// stays pending, and N-API reports it as uncaught.
static void call_done_cb(napi_env env, napi_value cb, void *context, void *data) {
	if (env == NULL || cb == NULL) {
		return;
	}
	napi_value undefined;
	if (napi_get_undefined(env, &undefined) != napi_ok) {
		return;
	}
	if (napi_call_function(env, undefined, cb, 0, NULL, NULL) != napi_ok) {
		return;
	}
}

static napi_value workspace_New_bridge(napi_env env, napi_callback_info info) {
	size_t argc = 1;
	napi_value on_created_cb;

	napi_get_cb_info(env, info, &argc, &on_created_cb, NULL, NULL);

	napi_value resource_name;
	napi_create_string_utf8(env, "golar: workspace created", NAPI_AUTO_LENGTH, &resource_name);

	napi_threadsafe_function cb_ptr;
	napi_create_threadsafe_function(env, on_created_cb, NULL, resource_name, 0, 1, NULL, NULL, NULL, call_done_cb, &cb_ptr);

	golar_js_workspace_New((uintptr_t)cb_ptr);
	return NULL;
}

static uint32_t get_thread_id_from_args(napi_env env, napi_callback_info info) {
	size_t argc = 1;
	napi_value thread_id_value;
	napi_get_cb_info(env, info, &argc, &thread_id_value, NULL, NULL);

	uint32_t thread_id;
	napi_get_value_uint32(env, thread_id_value, &thread_id);
	return thread_id;
}

static napi_value workspace_ReadRequestedFileAt_bridge(napi_env env, napi_callback_info info) {
	uint32_t thread_id = get_thread_id_from_args(env, info);
	golar_js_workspace_ReadRequestedFileAt(thread_id);
	return NULL;
}

static napi_value workspace_ReadFileById_bridge(napi_env env, napi_callback_info info) {
	uint32_t thread_id = get_thread_id_from_args(env, info);
	golar_js_workspace_ReadFileById(thread_id);
	return NULL;
}

static napi_value workspace_GetTypeAtLocation_bridge(napi_env env, napi_callback_info info) {
	uint32_t thread_id = get_thread_id_from_args(env, info);
	golar_js_workspace_GetTypeAtLocation(thread_id);
	return NULL;
}

static napi_value workspace_Lint_bridge(napi_env env, napi_callback_info info) {
	size_t argc = 1;
	napi_value done_cb;
	napi_get_cb_info(env, info, &argc, &done_cb, NULL, NULL);

	napi_value resource_name;
	napi_create_string_utf8(env, "golar: lint done", NAPI_AUTO_LENGTH, &resource_name);

	napi_threadsafe_function done_cb_tsfn;
	napi_create_threadsafe_function(env, done_cb, NULL, resource_name, 0, 1, NULL, NULL, NULL, call_done_cb, &done_cb_tsfn);

	golar_js_workspace_Lint((uintptr_t)done_cb_tsfn);
	return NULL;
}

static napi_value workspace_Report_bridge(napi_env env, napi_callback_info info) {
	uint32_t thread_id = get_thread_id_from_args(env, info);
	golar_js_workspace_Report(thread_id);
	return NULL;
}

static napi_value workspace_ReportCount_bridge(napi_env env, napi_callback_info info) {
	napi_value count;
	napi_create_uint32(env, golar_js_workspace_ReportCount(), &count);
	return count;
}

// The exception pending on env, cleared, or NULL when there is none.
static napi_value take_pending_exception(napi_env env) {
	bool pending = false;
	if (napi_is_exception_pending(env, &pending) != napi_ok || !pending) {
		return NULL;
	}
	napi_value exception;
	if (napi_get_and_clear_last_exception(env, &exception) != napi_ok) {
		return NULL;
	}
	return exception;
}

// Answers the Go side of a codegen request that JS could not answer, with
// reason (an exception or a rejection value, or NULL) as the file's service
// code error, so Go never waits for a response that will not come.
static void create_service_code_failed(napi_env env, uint32_t thread_id, napi_value reason, const char *fallback) {
	char *message = NULL;
	size_t message_len = 0;
	napi_value reason_str;
	if (reason != NULL &&
		napi_coerce_to_string(env, reason, &reason_str) == napi_ok &&
		napi_get_value_string_utf8(env, reason_str, NULL, 0, &message_len) == napi_ok) {
		message = malloc(message_len + 1);
		if (message != NULL &&
			napi_get_value_string_utf8(env, reason_str, message, message_len + 1, &message_len) != napi_ok) {
			free(message);
			message = NULL;
		}
	}
	// Coercing the reason can throw (a Symbol has no string conversion).
	take_pending_exception(env);

	if (message == NULL) {
		golar_js_jsCodegenCreateServiceCodeFailed(thread_id, (char *)fallback, strlen(fallback));
		return;
	}
	golar_js_jsCodegenCreateServiceCodeFailed(thread_id, message, message_len);
	free(message);
}

static napi_value create_service_code_fulfilled_cb(napi_env env, napi_callback_info info) {
	void *data;
	if (napi_get_cb_info(env, info, NULL, NULL, NULL, &data) != napi_ok) {
		return NULL;
	}
	golar_js_jsCodegenCreateServiceCodeResponse((uint32_t)(uintptr_t)data);
	return NULL;
}

static napi_value create_service_code_rejected_cb(napi_env env, napi_callback_info info) {
	size_t argc = 1;
	napi_value reason;
	void *data;
	if (napi_get_cb_info(env, info, &argc, &reason, NULL, &data) != napi_ok) {
		return NULL;
	}
	create_service_code_failed(env, (uint32_t)(uintptr_t)data, reason, "golar: createServiceCode was rejected");
	return NULL;
}

#define CREATE_SERVICE_CODE_CHECK(CALL, STEP) \
	if ((CALL) != napi_ok) { \
		create_service_code_failed(env, thread_id, take_pending_exception(env), "golar: createServiceCode failed: " STEP); \
		return; \
	}

static void call_create_service_code_cb(napi_env env, napi_value cb, void *context, void *data) {
	if (env == NULL || cb == NULL) {
		return;
	}
	uint32_t thread_id = (uint32_t)(uintptr_t)context;

	napi_value undefined;
	CREATE_SERVICE_CODE_CHECK(napi_get_undefined(env, &undefined), "napi_get_undefined");

	napi_value result;
	CREATE_SERVICE_CODE_CHECK(napi_call_function(env, undefined, cb, 0, NULL, &result), "the call threw");

	// TODO(perf): fast path for sync returns
	napi_value then;
	CREATE_SERVICE_CODE_CHECK(napi_get_named_property(env, result, "then", &then), "reading its result's then");

	napi_value callbacks[2];
	CREATE_SERVICE_CODE_CHECK(napi_create_function(env, "onFulfilled", NAPI_AUTO_LENGTH, create_service_code_fulfilled_cb, context, &callbacks[0]), "napi_create_function");
	CREATE_SERVICE_CODE_CHECK(napi_create_function(env, "onRejected", NAPI_AUTO_LENGTH, create_service_code_rejected_cb, context, &callbacks[1]), "napi_create_function");

	napi_value then_promise;
	CREATE_SERVICE_CODE_CHECK(napi_call_function(env, result, then, 2, callbacks, &then_promise), "its result is not a promise");
}

#undef CREATE_SERVICE_CODE_CHECK

static napi_value registerJsCodegen_bridge(napi_env env, napi_callback_info info) {
	size_t argc = 2;
	napi_value argv[argc];
	napi_get_cb_info(env, info, &argc, argv, NULL, NULL);

	uint32_t thread_id;
	napi_get_value_uint32(env, argv[0], &thread_id);

	napi_value resource_name;
	napi_create_string_utf8(env, "golar: create service code", NAPI_AUTO_LENGTH, &resource_name);

	napi_threadsafe_function create_service_code_tsfn;
	napi_create_threadsafe_function(env, argv[1], NULL, resource_name, 0, 1, NULL, NULL, (void *)(uintptr_t)thread_id, call_create_service_code_cb, &create_service_code_tsfn);

	golar_js_registerJsCodegen(thread_id, (uintptr_t)create_service_code_tsfn);

	return NULL;
}

static napi_value registerIpcCodegen_bridge(napi_env env, napi_callback_info info) {
	golar_js_registerIpcCodegen();
	return NULL;
}

// Calls a done callback with the exit code. An exception the callback throws
// stays pending, and N-API reports it as uncaught.
static void call_exit_code_cb(napi_env env, napi_value cb, void *context, void *data) {
	if (env == NULL || cb == NULL) {
		return;
	}
	napi_value undefined;
	if (napi_get_undefined(env, &undefined) != napi_ok) {
		return;
	}

	napi_value exit_code;
	if (napi_create_uint32(env, (uint32_t)(uintptr_t)data, &exit_code) != napi_ok) {
		return;
	}

	if (napi_call_function(env, undefined, cb, 1, &exit_code, NULL) != napi_ok) {
		return;
	}
}

static napi_value tsc_bridge(napi_env env, napi_callback_info info) {
	size_t argc = 2;
	napi_value argv[argc];
	napi_get_cb_info(env, info, &argc, argv, NULL, NULL);

	uint32_t thread_id;
	napi_get_value_uint32(env, argv[0], &thread_id);

	napi_value resource_name;
	napi_create_string_utf8(env, "golar: tsc done", NAPI_AUTO_LENGTH, &resource_name);

	napi_threadsafe_function done_cb_tsfn;
	napi_create_threadsafe_function(env, argv[1], NULL, resource_name, 0, 1, NULL, NULL, NULL, call_exit_code_cb, &done_cb_tsfn);

	golar_js_tsc(thread_id, (uintptr_t)done_cb_tsfn);

	return NULL;
}

static napi_value workspace_TypeCheck_bridge(napi_env env, napi_callback_info info) {
	size_t argc = 1;
	napi_value done_cb;
	napi_get_cb_info(env, info, &argc, &done_cb, NULL, NULL);

	napi_value resource_name;
	napi_create_string_utf8(env, "golar: typecheck done", NAPI_AUTO_LENGTH, &resource_name);

	napi_threadsafe_function done_cb_tsfn;
	napi_create_threadsafe_function(env, done_cb, NULL, resource_name, 0, 1, NULL, NULL, NULL, call_exit_code_cb, &done_cb_tsfn);

	golar_js_workspace_TypeCheck((uintptr_t)done_cb_tsfn);
	return NULL;
}

void napi_call_threadsafe_function_any(uintptr_t func, uintptr_t data, size_t is_blocking) {
	napi_call_threadsafe_function((napi_threadsafe_function)func, (void*)data, is_blocking);
}

NAPI_MODULE_INIT() {
	#ifdef GOLAR_NAPI_DYNAMIC
	if (!golar_init_napi_symbols()) {
		return NULL;
	}
	#endif

	napi_property_descriptor descriptors[] = {
		METHOD_DESCRIPTOR_STR("setSyncBuffer", setSyncBuffer_bridge),
		METHOD_DESCRIPTOR_STR("registerJsCodegen", registerJsCodegen_bridge),
		METHOD_DESCRIPTOR_STR("registerIpcCodegen", registerIpcCodegen_bridge),
		METHOD_DESCRIPTOR_STR("tsc", tsc_bridge),

		METHOD_DESCRIPTOR_STR("linter_RuleTesterLint", linter_RuleTesterLint_bridge),
		METHOD_DESCRIPTOR_STR("workspace_New", workspace_New_bridge),
		METHOD_DESCRIPTOR_STR("workspace_ReadRequestedFileAt", workspace_ReadRequestedFileAt_bridge),
		METHOD_DESCRIPTOR_STR("workspace_ReadFileById", workspace_ReadFileById_bridge),
		METHOD_DESCRIPTOR_STR("workspace_GetTypeAtLocation", workspace_GetTypeAtLocation_bridge),
		METHOD_DESCRIPTOR_STR("workspace_Lint", workspace_Lint_bridge),
		METHOD_DESCRIPTOR_STR("workspace_TypeCheck", workspace_TypeCheck_bridge),
		METHOD_DESCRIPTOR_STR("workspace_Report", workspace_Report_bridge),
		METHOD_DESCRIPTOR_STR("workspace_ReportCount", workspace_ReportCount_bridge),
	};

	napi_define_properties(env, exports, sizeof(descriptors)/sizeof(descriptors[0]), descriptors);
	return exports;
}
