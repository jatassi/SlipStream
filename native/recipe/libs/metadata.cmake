# Writes slipstream-native.json at the layer root. Every path in it is
# relative to the layer root, so the layer can be moved after it is built.

function(json_array out)
	set(arr "[]")
	set(i 0)
	foreach(item IN LISTS ARGN)
		string(JSON arr SET "${arr}" ${i} "\"${item}\"")
		math(EXPR i "${i} + 1")
	endforeach()
	set(${out} "${arr}" PARENT_SCOPE)
endfunction()

# libtorrent's ABI defines (TORRENT_NO_DEPRECATE and friends) must match
# between its own build and every translation unit that includes its headers.
file(STRINGS ${LAYER_PREFIX}/lib/pkgconfig/libtorrent-rasterbar.pc cflags REGEX "^Cflags:")
string(REGEX MATCHALL "-D[^ ]+" defines "${cflags}")
list(TRANSFORM defines REPLACE "^-D" "")
list(PREPEND defines SLIPSTREAM_NATIVE_BUNDLE)

json_array(include_dirs include)
json_array(defines_json ${defines})
json_array(static_libs lib/libtorrent-rasterbar.a lib/libssl.a lib/libcrypto.a)
json_array(system_libs ${SYSTEM_LIBS})

set(json "{}")
string(JSON json SET "${json}" compiler "\"${COMPILER}\"")
string(JSON json SET "${json}" include_dirs "${include_dirs}")
string(JSON json SET "${json}" defines "${defines_json}")
string(JSON json SET "${json}" static_libs "${static_libs}")
string(JSON json SET "${json}" system_libs "${system_libs}")
file(WRITE ${LAYER_PREFIX}/slipstream-native.json "${json}\n")
