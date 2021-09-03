#include <vector>
#include <string>

#include <windows.h>
#include <shlwapi.h>

#include <Commctrl.h>
#include <commoncontrols.h>
#include <comdef.h>
#include <Gdiplus.h>

#define EXPORT __declspec(dllexport)

// functions will be declared later
bool Init();
bool UnInit();
std::string GetBinaryIconBase64Png(const std::wstring binaryPath);

extern "C"
{
    EXPORT DWORD _cdecl BinaryIconReaderInit()
    {
        return Init();
    }

    EXPORT DWORD _cdecl BinaryIconReaderUnInit()
    {
        return UnInit();
    }

    EXPORT DWORD _cdecl BinaryIconReaderReadBase64Png(const wchar_t* binaryPath, unsigned char* buff, DWORD* _in_out_buffSize)
    {
        DWORD inBufSize = *_in_out_buffSize;

        std::string base64 = GetBinaryIconBase64Png(binaryPath);
        if (base64.length() == 0)
        {
            *_in_out_buffSize = 0;
            return false;
        }
        
        *_in_out_buffSize = static_cast<DWORD>(base64.length());

        if (base64.length() > inBufSize)
            return false;

        if (memcpy_s(buff, inBufSize, base64.c_str(), base64.length()))
            return false;

        return true;
    }
}

static const std::string base64_chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
std::string base64_encode(unsigned char const* bytes_to_encode, size_t in_len) {
    std::string ret;
    int i = 0;
    int j = 0;
    unsigned char char_array_3[3];
    unsigned char char_array_4[4];

    while (in_len--) {
        char_array_3[i++] = *(bytes_to_encode++);
        if (i == 3) {
            char_array_4[0] = (char_array_3[0] & 0xfc) >> 2;
            char_array_4[1] = ((char_array_3[0] & 0x03) << 4) + ((char_array_3[1] & 0xf0) >> 4);
            char_array_4[2] = ((char_array_3[1] & 0x0f) << 2) + ((char_array_3[2] & 0xc0) >> 6);
            char_array_4[3] = char_array_3[2] & 0x3f;

            for (i = 0; (i < 4); i++)
                ret += base64_chars[char_array_4[i]];
            i = 0;
        }
    }

    if (i)
    {
        for (j = i; j < 3; j++)
            char_array_3[j] = '\0';

        char_array_4[0] = (char_array_3[0] & 0xfc) >> 2;
        char_array_4[1] = ((char_array_3[0] & 0x03) << 4) + ((char_array_3[1] & 0xf0) >> 4);
        char_array_4[2] = ((char_array_3[1] & 0x0f) << 2) + ((char_array_3[2] & 0xc0) >> 6);
        char_array_4[3] = char_array_3[2] & 0x3f;

        for (j = 0; (j < i + 1); j++)
            ret += base64_chars[char_array_4[j]];

        while ((i++ < 3))
            ret += '=';

    }

    return ret;
}

// http://msdn.microsoft.com/en-us/library/windows/desktop/ms533843(v=vs.85).aspx
int getEncoderClsid(const WCHAR* format, CLSID* pClsid)
{
    UINT  num = 0;          // number of image encoders
    UINT  size = 0;         // size of the image encoder array in bytes

    Gdiplus::ImageCodecInfo* pImageCodecInfo = NULL;

    if (Gdiplus::Status::Ok != Gdiplus::GetImageEncodersSize(&num, &size))
        return -1;  // Failure
    if (size == 0)
        return -1;  // Failure

    pImageCodecInfo = (Gdiplus::ImageCodecInfo*)(malloc(size));
    if (pImageCodecInfo == NULL)
        return -1;  // Failure

    if (Gdiplus::Status::Ok != GetImageEncoders(num, size, pImageCodecInfo))
        return -1;

    for (UINT j = 0; j < num; ++j)
    {
        if (wcscmp(pImageCodecInfo[j].MimeType, format) == 0)
        {
            *pClsid = pImageCodecInfo[j].Clsid;
            free(pImageCodecInfo);
            return j;  // Success
        }
    }

    free(pImageCodecInfo);
    return -1;  // Failure
}

_COM_SMARTPTR_TYPEDEF(IImageList, __uuidof(IImageList));
static HICON getShellIcon(int shilSize, const std::wstring& fname)
{
    SHFILEINFO fi = { 0 };
    HICON hIcon = NULL;
        
    if (SHGetFileInfo(fname.c_str(), 0, &fi, sizeof(fi), SHGFI_SYSICONINDEX) != 0)
    {
        IImageListPtr spiml;
        if (S_OK != SHGetImageList(shilSize, IID_PPV_ARGS(&spiml)))
            return NULL;

        spiml->GetIcon(fi.iIcon, ILD_TRANSPARENT, &hIcon); // to check for S_OK?
    }
    else
        return NULL;

    return hIcon;
}

struct BITMAP_AND_BYTES
{
    Gdiplus::Bitmap* bmp;
    int32_t* bytes;
};

static BITMAP_AND_BYTES createAlphaChannelBitmapFromIcon(HICON hIcon)
{
    int32_t* colorBits = NULL;
    BITMAP bm = { 0 };

    // Get the icon info
    ICONINFO iconInfo = { 0 };
    if (GetIconInfo(hIcon, &iconInfo) == false)
        return { 0 };

    bool isSuccess = false;
    // Get the screen DC
    HDC dc = GetDC(NULL);
    do {
        if (dc == NULL)
            break;

        // Get icon size info

        if (!GetObject(iconInfo.hbmColor, sizeof(BITMAP), &bm))
            break;

        // Set up BITMAPINFO
        BITMAPINFO bmi = { 0 };
        bmi.bmiHeader.biSize = sizeof(BITMAPINFOHEADER);
        bmi.bmiHeader.biWidth = bm.bmWidth;
        bmi.bmiHeader.biHeight = -bm.bmHeight;
        bmi.bmiHeader.biPlanes = 1;
        bmi.bmiHeader.biBitCount = 32;
        bmi.bmiHeader.biCompression = BI_RGB;

        // Extract the color bitmap
        int nBits = bm.bmWidth * bm.bmHeight;
        colorBits = new (std::nothrow) int32_t[nBits];
        if (!colorBits)
            break;

        if (!GetDIBits(dc, iconInfo.hbmColor, 0, bm.bmHeight, colorBits, &bmi, DIB_RGB_COLORS))
        {
            delete[] colorBits;
            break;
        }

        // Check whether the color bitmap has an alpha channel
        BOOL hasAlpha = FALSE;
        for (int i = 0; i < nBits; i++)
        {
            if ((colorBits[i] & 0xff000000) != 0)
            {
                hasAlpha = TRUE;
                break;
            }
        }

        // If no alpha values available, apply the mask bitmap
        if (!hasAlpha)
        {
            // Extract the mask bitmap
            int32_t* maskBits = new (std::nothrow) int32_t[nBits];
            if (!maskBits)
                break;

            if (!GetDIBits(dc, iconInfo.hbmMask, 0, bm.bmHeight, maskBits, &bmi, DIB_RGB_COLORS))
            {
                delete[] maskBits;
                break;
            }

            // Copy the mask alphas into the color bits
            for (int i = 0; i < nBits; i++)
            {
                if (maskBits[i] == 0)
                    colorBits[i] |= 0xff000000;
            }
            delete[] maskBits;
        }

        isSuccess = true;
    } while (false);

    // Release DC and GDI bitmaps
    if (dc)
        ReleaseDC(NULL, dc);
    ::DeleteObject(iconInfo.hbmColor);
    ::DeleteObject(iconInfo.hbmMask);

    if (!isSuccess)
        return { 0 };

    // Create GDI+ Bitmap
    Gdiplus::Bitmap* bmp = new Gdiplus::Bitmap(bm.bmWidth, bm.bmHeight, bm.bmWidth * 4, PixelFormat32bppARGB, (BYTE*)colorBits);
    return { bmp, colorBits };
}

bool savePngMemory(Gdiplus::Bitmap* gdiBitmap, const CLSID& encoderClsid, std::vector<BYTE>& data)
{
    IStream* istream = nullptr;

    bool isSuccess = false;
    do {
        if (S_OK != CreateStreamOnHGlobal(NULL, TRUE, &istream))
            break;

        Gdiplus::Status status = gdiBitmap->Save(istream, &encoderClsid);
        if (status != Gdiplus::Status::Ok)
            break;

        //get memory handle associated with istream
        HGLOBAL hg = NULL;
        if (S_OK != GetHGlobalFromStream(istream, &hg))
            break;

        //copy IStream to buffer
        SIZE_T bufsize = GlobalSize(hg);
        if (bufsize <= 0)
            break;

        data.resize(bufsize);

        //lock & unlock memory
        LPVOID pimage = GlobalLock(hg);
        if (pimage == NULL)
            break;

        memcpy(&data[0], pimage, bufsize);
        GlobalUnlock(hg);

        isSuccess = true;
    } while (false);

    if (istream)
        istream->Release();

    return isSuccess;
}

static bool extractBinaryIconToBase64Png(const int shilsize, const CLSID& encoderClsid, const std::wstring& binaryPath, std::string& outBase64PngStr)
{
    HICON hIcon = getShellIcon(shilsize, binaryPath);
    if (hIcon == NULL)
        return false;

    BITMAP_AND_BYTES bbs = createAlphaChannelBitmapFromIcon(hIcon);

    bool isSuccess = false;
    do
    {
        if (!bbs.bmp || !bbs.bytes)
            break;

        std::vector<BYTE> data;

        if (!savePngMemory(bbs.bmp, encoderClsid, data))
            break;

        if (data.empty())
            break;
        outBase64PngStr = base64_encode(&data[0], data.size());

        isSuccess = true;
    } while (false);

    if (bbs.bmp)
        delete bbs.bmp;
    if (bbs.bytes)
        delete[] bbs.bytes;
    if (hIcon)
        DestroyIcon(hIcon);

    return isSuccess;
}

static CLSID _encoderClsid = GUID_NULL;
static ULONG_PTR _gdiPlusToken = 0;

bool UnInit()
{
    if (!_gdiPlusToken)
        return false;

    Gdiplus::GdiplusShutdown(_gdiPlusToken);
    _gdiPlusToken = 0;

    return true;
}

bool Init()
{
    Gdiplus::GdiplusStartupInput gdiplusStartupInput = { 0 };
    if (Gdiplus::Status::Ok != Gdiplus::GdiplusStartup(&_gdiPlusToken, &gdiplusStartupInput, NULL))
    {
        UnInit();
        return false;
    }

    if (getEncoderClsid(L"image/png", &_encoderClsid) < 0)
    {
        UnInit();
        return false;
    }

    return true;
}

std::string GetBinaryIconBase64Png(const std::wstring binaryPath)
{
    if (!_gdiPlusToken)
        return "";

    std::string base64Png;
    extractBinaryIconToBase64Png(SHIL_LARGE, _encoderClsid, binaryPath, base64Png);
    return base64Png;
}