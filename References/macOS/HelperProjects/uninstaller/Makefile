
#In case of compilation errors, please, check if "Xcode Command Line Tools" are installed
#(to install it's latest version - use command: xcode-select --install)

# you should install and use the cross-development SDK (Command Line Tools) for target (10.10?) version of OS X
all:
	rm -fr bin
	mkdir bin

	cc -m64 -framework Foundation \
		-D IS_INSTALLER=0 \
		-framework ServiceManagement \
		-framework Security \
		-Xlinker -sectcreate -Xlinker __TEXT -Xlinker __info_plist -Xlinker "IVPN Uninstaller-Info.plist" \
		uninstaller.c \
		-o "bin/IVPN Uninstaller"

	cc -m64 -framework Foundation \
		-D IS_INSTALLER=1 \
		-framework ServiceManagement \
		-framework Security \
		-Xlinker -sectcreate -Xlinker __TEXT -Xlinker __info_plist -Xlinker "IVPN Installer-Info.plist" \
		uninstaller.c \
		-o "bin/IVPN Installer"
