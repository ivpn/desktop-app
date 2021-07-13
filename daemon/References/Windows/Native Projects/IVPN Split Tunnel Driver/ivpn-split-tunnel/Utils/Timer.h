#pragma once

#include "../Driver.h"

namespace utils
{

#ifdef DEBUG
	/// <summary>
	/// ElapsedTimePrintDubugger calculate and print elapsed time in current block
	/// Do nothing in release mode
	/// </summary>
	class ElapsedTimePrintDubugger {
	public:
		ElapsedTimePrintDubugger(const LPCSTR blockName);
		ElapsedTimePrintDubugger(const LPCSTR blockName, LONGLONG mcsThreshold) : ElapsedTimePrintDubugger(blockName)
		{
			_mcsThresholdToPrint = mcsThreshold;
		};

		~ElapsedTimePrintDubugger();
	private:
		LPCSTR _blockName;
		LARGE_INTEGER _startTime;

		LONGLONG _mcsThresholdToPrint;
	};

#define DEBUG_PrintElapsedTime() auto __dbg_timer__ = utils::ElapsedTimePrintDubugger(__FUNCTION__)
#define DEBUG_PrintElapsedTimeEx(microseconds) auto __dbg_timer__ = utils::ElapsedTimePrintDubugger(__FUNCTION__, microseconds)

#else // RELEASE

#define DEBUG_PrintElapsedTime() 
#define DEBUG_PrintElapsedTimeEx(microseconds) 

#endif // RELEASE

} // namespace utils