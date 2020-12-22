
#In case of compilation errors, please, check if "Xcode Command Line Tools" are installed
#(to install it's latest version - use command: xcode-select --install)

# you should install and use the cross-development SDK (Command Line Tools) for target (10.10?) version of OS X
all:
#		cc -v -m64 -dynamiclib
		cc -m64 -dynamiclib \
			libivpn.c power_change_notifications.c \
			-mmacosx-version-min=10.10 \
			-current_version 1.0 \
			-compatibility_version 1.0 \
			-fvisibility=hidden \
			-flat_namespace \
			-framework Foundation \
			-framework ServiceManagement \
			-framework SystemConfiguration \
			-framework IOKit \
			-framework Security \
			-o libivpn.dylib
