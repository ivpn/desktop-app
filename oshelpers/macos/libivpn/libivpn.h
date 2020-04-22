
#include <stdio.h>
#include <stdlib.h>
#include <dlfcn.h>

#define SUCCESS 0
#define ERROR_LIB_NOT_FOUND -1
#define ERROR_METHOD_NOT_FOUND -2

void UnLoadLibrary();
int start_xpc_listener(char *name, int serviceTcpPort, uint64_t secret);