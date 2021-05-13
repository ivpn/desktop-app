#include "libivpn.h"

#define LIBIVPN_PATH "/Applications/IVPN.app/Contents/MonoBundle/libivpn.dylib"
#define FUNC_start_xpc_listener "start_xpc_listener"

#define LOG_ERR_PREFIX "ERROR (helpers.h)"

void *lib = NULL;
void (*func_start_xpc_listener)(char *name, int serviceTcpPort, uint64_t secret) = NULL;

void UnLoadLibrary()
{
	func_start_xpc_listener = NULL;

	if (lib != NULL)
		dlclose(lib);
}

int LoadLibrary()
{
	int ret = SUCCESS;
	if (lib != NULL)
		return ret;

  	lib = dlopen(LIBIVPN_PATH, RTLD_LAZY);
	if ( lib == NULL )
	{
		printf("%s: dlopen() failed. '%s' not found\n", LOG_ERR_PREFIX, LIBIVPN_PATH);
		return ERROR_LIB_NOT_FOUND;
	}

	func_start_xpc_listener = dlsym(lib, FUNC_start_xpc_listener);
	if (func_start_xpc_listener==NULL)
	{
		printf("%s: dlsym() failed. Method '%s' not found in '%s' \n", LOG_ERR_PREFIX, FUNC_start_xpc_listener, LIBIVPN_PATH);
		ret = ERROR_METHOD_NOT_FOUND;
	}

	if (ret != SUCCESS)
		UnLoadLibrary();

  	return ret;
}

int start_xpc_listener(char *name, int serviceTcpPort, uint64_t secret)
{
	int loadErr = LoadLibrary();
	if (loadErr != SUCCESS)
		return loadErr;

	func_start_xpc_listener(name, serviceTcpPort, secret);
	return SUCCESS;
}
