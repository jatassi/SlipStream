# libtorrent includes its bundled copy of Apple's net/route.h, under the
# APSL and the BSD-4-Clause advertising clause, whenever TARGET_OS_IPHONE is
# defined, and TargetConditionals.h defines it on macOS too, as 0. Testing
# its value makes macOS use the system <net/route.h>. Deleting the copy makes
# any build that still includes it fail, and keeps it out of the layer.

set(enum_net ${SOURCE_DIR}/src/enum_net.cpp)
set(old "#if defined TARGET_IPHONE_SIMULATOR || defined TARGET_OS_IPHONE")
file(READ ${enum_net} source)
string(FIND "${source}" "${old}" at)
if(at EQUAL -1)
	message(FATAL_ERROR "${enum_net} no longer has the line `${old}`; update this patch")
endif()
string(REPLACE "${old}" "#if TARGET_OS_IPHONE" source "${source}")
file(WRITE ${enum_net} "${source}")
file(REMOVE ${SOURCE_DIR}/include/libtorrent/aux_/route.h)
