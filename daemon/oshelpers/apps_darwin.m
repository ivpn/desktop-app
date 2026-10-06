//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2026 IVPN Limited.
//
//  This file is part of the Daemon for IVPN Client Desktop.
//
//  The Daemon for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The Daemon for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the Daemon for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

// Implementation for the C functions declared in apps_darwin.go's cgo
// preamble - kept in a separate .m file (matching the existing
// wifiNotifier/darwin/obsolete/src.m pattern) rather than inline in the Go
// file's preamble, since ObjC syntax highlighting/tooling doesn't apply to Go
// string comments.

#include <stdlib.h>
#include <string.h>
#import <AppKit/AppKit.h>

int app_bundle_info(const char *bundlePath, char **outDisplayName) {
    if (outDisplayName) *outDisplayName = NULL;
    if (!bundlePath) return 1;

    @autoreleasepool {
        NSString *path = [NSString stringWithUTF8String:bundlePath];
        NSBundle *bundle = [NSBundle bundleWithPath:path];
        if (!bundle) return 1;

        NSString *displayName = [bundle objectForInfoDictionaryKey:@"CFBundleDisplayName"];
        if (displayName.length == 0) displayName = [bundle objectForInfoDictionaryKey:@"CFBundleName"];
        if (displayName.length == 0) displayName = [[path lastPathComponent] stringByDeletingPathExtension];
        if (outDisplayName) *outDisplayName = strdup(displayName.UTF8String);
        return 0;
    }
}

int app_icon_png(const char *bundlePath, int maxSizePx, unsigned char **outData, long *outLen) {
    if (outData) *outData = NULL;
    if (outLen) *outLen = 0;
    if (!bundlePath || maxSizePx <= 0) return 1;

    @autoreleasepool {
        NSString *path = [NSString stringWithUTF8String:bundlePath];
        NSImage *icon = [[NSWorkspace sharedWorkspace] iconForFile:path];
        if (!icon) return 1;

        // Rendered into an offscreen bitmap on purpose: -[NSImage lockFocus] needs a
        // window-server connection, which the daemon (root LaunchDaemon, no GUI
        // session) does not have.
        NSBitmapImageRep *rep = [[NSBitmapImageRep alloc] initWithBitmapDataPlanes:NULL
                                                                       pixelsWide:maxSizePx
                                                                       pixelsHigh:maxSizePx
                                                                    bitsPerSample:8
                                                                  samplesPerPixel:4
                                                                         hasAlpha:YES
                                                                         isPlanar:NO
                                                                   colorSpaceName:NSDeviceRGBColorSpace
                                                                      bytesPerRow:0
                                                                     bitsPerPixel:0];
        if (!rep) return 1;
        rep.size = NSMakeSize(maxSizePx, maxSizePx);

        NSGraphicsContext *context = [NSGraphicsContext graphicsContextWithBitmapImageRep:rep];
        if (!context) return 1;

        [NSGraphicsContext saveGraphicsState];
        [NSGraphicsContext setCurrentContext:context];
        [icon drawInRect:NSMakeRect(0, 0, maxSizePx, maxSizePx)
                fromRect:NSZeroRect
               operation:NSCompositingOperationSourceOver
                fraction:1.0];
        [context flushGraphics];
        [NSGraphicsContext restoreGraphicsState];

        NSData *png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
        if (!png || png.length == 0) return 1;

        void *buf = malloc(png.length);
        if (!buf) return 1;
        memcpy(buf, png.bytes, png.length);
        if (outData) *outData = (unsigned char *)buf;
        if (outLen) *outLen = (long)png.length;
        return 0;
    }
}

void app_free(void *ptr) {
    free(ptr);
}
