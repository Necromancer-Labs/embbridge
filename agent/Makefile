# embbridge agent - simple local build
#
# Usage:
#   make          Build debug version
#   make release  Build release version
#   make clean    Clean build artifacts

CC = gcc
CFLAGS = -Wall -Wextra -std=c99 -I./include
LDFLAGS =

# Source files
SRCS = src/main.c \
       src/transport.c \
       src/protocol.c \
       src/commands/cmd_dispatch.c \
       src/commands/helpers.c \
       src/commands/basic_commands.c \
       src/commands/file_transfer.c \
       src/commands/file_operations.c \
       src/commands/system_commands.c

TARGET = edb-agent

# Debug by default
CFLAGS += -g -DDEBUG

.PHONY: all release clean

all: $(TARGET)

release: CFLAGS = -Wall -Wextra -std=c99 -I./include -Os -DNDEBUG
release: LDFLAGS = -s
release: $(TARGET)

$(TARGET): $(SRCS)
	$(CC) $(CFLAGS) $(LDFLAGS) -o $@ $(SRCS)

clean:
	rm -f $(TARGET)
