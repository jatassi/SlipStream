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

# On Windows, libtorrent replaces the store set_default_verify_paths filled
# with one built from the system ROOT store, so the CA that SSL_CERT_FILE
# names goes untrusted there while Linux and macOS trust it. The new store
# loads that file too, but not OpenSSL's built-in default paths, which on
# Windows can name a folder an unprivileged user can create. It reads the
# Win32 environment, because on Windows Go's os.Setenv leaves the C
# runtime's copy that getenv reads unchanged.

set(session_impl ${SOURCE_DIR}/src/session_impl.cpp)
set(old [=[
		SSL_CTX* ssl_ctx = m_ssl_ctx.native_handle();
		SSL_CTX_set_cert_store(ssl_ctx, store);
]=])
set(new [=[
		DWORD const cert_file_size = GetEnvironmentVariableA("SSL_CERT_FILE", nullptr, 0);
		if (store && cert_file_size > 1)
		{
			std::string cert_file(cert_file_size, '\0');
			GetEnvironmentVariableA("SSL_CERT_FILE", &cert_file[0], cert_file_size);
			X509_STORE_load_file(store, cert_file.c_str());
		}

		SSL_CTX* ssl_ctx = m_ssl_ctx.native_handle();
		SSL_CTX_set_cert_store(ssl_ctx, store);
]=])
file(READ ${session_impl} source)
string(FIND "${source}" "${old}" at)
if(at EQUAL -1)
	message(FATAL_ERROR "${session_impl} no longer has the lines `${old}`; update this patch")
endif()
string(REPLACE "${old}" "${new}" source "${source}")
file(WRITE ${session_impl} "${source}")
