# Builds a static-only GCC cross toolchain for TRIPLE against musl into
# LAYER_PREFIX. It follows musl-cross-make: GCC is configured once, musl is
# built with the compiler from GCC's build tree, then GCC finishes its target
# libraries against that musl. Every program the layer holds is linked
# statically, so the layer runs on any Linux host of its architecture, and
# its sysroot sits under its prefix, so the layer runs from any directory.
#
# Run it through `release deps build -kind toolchain`.

cmake_minimum_required(VERSION 3.24)

foreach(var LAYER_PREFIX WORK_DIR DOWNLOAD_DIR TRIPLE
		BINUTILS_URL BINUTILS_SHA256 GCC_URL GCC_SHA256 GMP_URL GMP_SHA256
		MPFR_URL MPFR_SHA256 MPC_URL MPC_SHA256 MUSL_URL MUSL_SHA256
		LINUX_HEADERS_URL LINUX_HEADERS_SHA256)
	if(NOT DEFINED ${var})
		message(FATAL_ERROR "${var} is required; run the recipe through `release deps build`")
	endif()
endforeach()

cmake_host_system_information(RESULT jobs QUERY NUMBER_OF_LOGICAL_CORES)
string(REGEX MATCH "^[^-]+" arch ${TRIPLE})
set(sysroot /${TRIPLE})
set(build_sysroot ${WORK_DIR}/sysroot)

# source(<name> <var>) downloads the pinned archive once into DOWNLOAD_DIR,
# checks its SHA-256, extracts it and sets <var> to its top directory.
function(source name var)
	string(TOUPPER ${name} id)
	string(REPLACE "-" "_" id ${id})
	cmake_path(GET ${id}_URL FILENAME file)
	set(archive ${DOWNLOAD_DIR}/${file})
	if(EXISTS ${archive})
		file(SHA256 ${archive} sum)
	endif()
	if(NOT sum STREQUAL ${id}_SHA256)
		file(DOWNLOAD ${${id}_URL} ${archive} EXPECTED_HASH SHA256=${${id}_SHA256} TLS_VERIFY ON)
	endif()
	file(ARCHIVE_EXTRACT INPUT ${archive} DESTINATION ${WORK_DIR}/src/${name})
	file(GLOB top LIST_DIRECTORIES true ${WORK_DIR}/src/${name}/*)
	set(${var} ${top} PARENT_SCOPE)
endfunction()

function(run dir)
	file(MAKE_DIRECTORY ${dir})
	execute_process(COMMAND ${ARGN} WORKING_DIRECTORY ${dir} COMMAND_ERROR_IS_FATAL ANY)
endfunction()

foreach(name binutils gcc gmp mpfr mpc musl linux-headers)
	source(${name} src_${name})
endforeach()
foreach(lib gmp mpfr mpc)
	file(RENAME ${src_${lib}} ${src_gcc}/${lib})
endforeach()

set(host_flags CFLAGS=-O2 CXXFLAGS=-O2 "LDFLAGS=-s -static")
set(common --target=${TRIPLE} --prefix= --libdir=/lib --with-sysroot=${sysroot}
	--disable-multilib --disable-nls --disable-werror)

set(binutils ${WORK_DIR}/binutils)
run(${binutils} ${src_binutils}/configure ${common} ${host_flags}
	--disable-separate-code --enable-deterministic-archives
	--disable-gprofng --disable-plugins)
run(${binutils} make -j${jobs})

# GCC's fixincludes step needs the build sysroot's header directory to exist
# before musl installs into it.
file(MAKE_DIRECTORY ${build_sysroot}/include)
set(gcc ${WORK_DIR}/gcc)
run(${gcc} ${src_gcc}/configure ${common} ${host_flags}
	CFLAGS_FOR_TARGET=-O2 CXXFLAGS_FOR_TARGET=-O2
	--with-build-sysroot=${build_sysroot}
	--with-native-system-header-dir=/include
	--enable-languages=c,c++
	--disable-shared --disable-bootstrap --disable-assembly
	--disable-lto --disable-plugin
	--enable-tls --enable-initfini-array --enable-libstdcxx-time=rt
	--disable-libstdcxx-pch --disable-libsanitizer --disable-libgomp
	--disable-libquadmath --disable-libitm --disable-libvtv --disable-libssp
	--disable-gnu-indirect-function --disable-decimal-float --disable-fixed-point
	AR_FOR_TARGET=${binutils}/binutils/ar
	AS_FOR_TARGET=${binutils}/gas/as-new
	LD_FOR_TARGET=${binutils}/ld/ld-new
	NM_FOR_TARGET=${binutils}/binutils/nm-new
	OBJCOPY_FOR_TARGET=${binutils}/binutils/objcopy
	OBJDUMP_FOR_TARGET=${binutils}/binutils/objdump
	RANLIB_FOR_TARGET=${binutils}/binutils/ranlib
	READELF_FOR_TARGET=${binutils}/binutils/readelf
	STRIP_FOR_TARGET=${binutils}/binutils/strip-new)
run(${gcc} make -j${jobs} all-gcc)

set(musl ${WORK_DIR}/musl)
set(musl_vars "CC=${gcc}/gcc/xgcc -B ${gcc}/gcc"
	LIBCC=${gcc}/${TRIPLE}/libgcc/libgcc.a
	AR=${binutils}/binutils/ar
	RANLIB=${binutils}/binutils/ranlib)
run(${musl} ${src_musl}/configure --prefix= --host=${TRIPLE} --disable-shared ${musl_vars})
run(${musl} make DESTDIR=${build_sysroot} install-headers)
run(${src_linux-headers} make ARCH=${arch} prefix= DESTDIR=${build_sysroot} install)
run(${gcc} make -j${jobs} all-target-libgcc)
run(${musl} make -j${jobs})
run(${musl} make DESTDIR=${build_sysroot} install-libs)
run(${gcc} make -j${jobs})

run(${binutils} make DESTDIR=${LAYER_PREFIX} install)
run(${gcc} make DESTDIR=${LAYER_PREFIX} install)
run(${musl} make DESTDIR=${LAYER_PREFIX}${sysroot} install)
run(${src_linux-headers} make ARCH=${arch} prefix= DESTDIR=${LAYER_PREFIX}${sysroot} install)
file(REMOVE_RECURSE ${LAYER_PREFIX}/share)

file(READ ${src_musl}/COPYRIGHT musl_license)
file(READ ${src_gcc}/COPYING3 gpl3)
file(READ ${src_gcc}/COPYING.RUNTIME gcc_exception)
file(WRITE ${LAYER_PREFIX}/licenses/musl.txt "${musl_license}")
foreach(runtime libgcc libstdc++)
	file(WRITE ${LAYER_PREFIX}/licenses/${runtime}.txt "${gpl3}\n${gcc_exception}")
endforeach()
file(CONFIGURE OUTPUT ${LAYER_PREFIX}/slipstream-native.json
	CONTENT "{\"cc\": \"bin/${TRIPLE}-gcc\", \"cxx\": \"bin/${TRIPLE}-g++\"}\n")
