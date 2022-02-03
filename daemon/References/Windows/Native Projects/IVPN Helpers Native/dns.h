#pragma once

#include <windows.h>

#include <algorithm>
#include <string> 

enum class Operation
{
	Set = 0,
	Add = 1,
	Remove = 2
};

inline void toLowerStr(std::string* str) {
	std::transform((*str).begin(), (*str).end(), (*str).begin(), [](unsigned char c) { return std::tolower(c); });
}
inline void toLowerWStr(std::wstring* str) {
	std::transform((*str).begin(), (*str).end(), (*str).begin(), ::tolower);
}