cmake_minimum_required(VERSION 3.10)
project(edb-agent C)

# C99 standard
set(CMAKE_C_STANDARD 99)
set(CMAKE_C_STANDARD_REQUIRED ON)

# Compiler flags
set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -Wall -Wextra -Werror -Wno-unused-function")
set(CMAKE_C_FLAGS_RELEASE "-Os -DNDEBUG")
set(CMAKE_C_FLAGS_DEBUG "-g -DDEBUG")

# Strip release builds for smaller size
if(CMAKE_BUILD_TYPE STREQUAL "Release")
    set(CMAKE_EXE_LINKER_FLAGS "${CMAKE_EXE_LINKER_FLAGS} -s")
endif()

# Static linking for portability on embedded systems
set(CMAKE_EXE_LINKER_FLAGS "${CMAKE_EXE_LINKER_FLAGS} -static")

# Source files
set(SOURCES
    src/main.c
    src/transport.c
    src/protocol.c
    src/commands.c
)

# Include directories
include_directories(include)

# Build the agent binary
add_executable(edb-agent ${SOURCES})

# Install target
install(TARGETS edb-agent RUNTIME DESTINATION bin)
