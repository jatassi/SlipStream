# Mirrors the pinned llvm-mingw release into LAYER_PREFIX as released,
# without the sysroots of the architectures other than TRIPLE's.
#
# Run it through `release deps build -kind toolchain`.

cmake_minimum_required(VERSION 3.24)

foreach(var LAYER_PREFIX WORK_DIR DOWNLOAD_DIR TRIPLE LLVM_MINGW_URL LLVM_MINGW_SHA256)
	if(NOT DEFINED ${var})
		message(FATAL_ERROR "${var} is required; run the recipe through `release deps build`")
	endif()
endforeach()

cmake_path(GET LLVM_MINGW_URL FILENAME file)
set(archive ${DOWNLOAD_DIR}/${file})
if(EXISTS ${archive})
	file(SHA256 ${archive} sum)
endif()
if(NOT sum STREQUAL LLVM_MINGW_SHA256)
	file(DOWNLOAD ${LLVM_MINGW_URL} ${archive} EXPECTED_HASH SHA256=${LLVM_MINGW_SHA256} TLS_VERIFY ON)
endif()
file(ARCHIVE_EXTRACT INPUT ${archive} DESTINATION ${WORK_DIR})
file(GLOB top LIST_DIRECTORIES true ${WORK_DIR}/*)
file(RENAME ${top} ${LAYER_PREFIX})

file(GLOB sysroots LIST_DIRECTORIES true ${LAYER_PREFIX}/*-w64-mingw32)
list(REMOVE_ITEM sysroots ${LAYER_PREFIX}/${TRIPLE})
file(REMOVE_RECURSE ${sysroots})

file(CONFIGURE OUTPUT ${LAYER_PREFIX}/slipstream-native.json
	CONTENT "{\"cc\": \"bin/${TRIPLE}-clang.exe\", \"cxx\": \"bin/${TRIPLE}-clang++.exe\"}\n")
