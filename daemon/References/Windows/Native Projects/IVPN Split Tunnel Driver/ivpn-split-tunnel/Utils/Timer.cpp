#include <ntifs.h>

#include "Timer.h"
#include "Timer.tmh"

namespace utils
{ 
#ifdef DEBUG

ElapsedTimePrintDubugger::ElapsedTimePrintDubugger(const LPCSTR blockName) 
{
	_mcsThresholdToPrint = 0;
	_blockName = blockName;
	_startTime = KeQueryPerformanceCounter(NULL);
}


ElapsedTimePrintDubugger::~ElapsedTimePrintDubugger()
{
	LARGE_INTEGER frequency;
	LARGE_INTEGER stopTime = KeQueryPerformanceCounter(&frequency);
	LONGLONG elapsed = stopTime.QuadPart - _startTime.QuadPart;
	if (elapsed>0)
	{	
		elapsed = (elapsed * 1000000) / frequency.QuadPart;

		if (_mcsThresholdToPrint == 0 || elapsed > _mcsThresholdToPrint )
		{
			TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%s) TIMER elapsed: %I64u µs (%I64u ms.).", _blockName, elapsed, elapsed / 1000);
		}
	}
}
#endif // DEBUG

} // namespace utils