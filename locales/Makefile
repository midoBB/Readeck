# SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
#
# SPDX-License-Identifier: AGPL-3.0-only

UV=$(shell command -pv uv)

all: compile

# Extract all the messages
extract:
ifneq ($(UV),)
	uv run messages.py extract
else
	@echo "ERROR: uv not found"
	@false
endif

# Update po files
update:
ifneq ($(UV),)
	uv run messages.py update
else
	@echo "ERROR: uv not found"
	@false
endif

# Compile po to mo files
compile:
ifneq ($(UV),)
	$(UV) run messages.py compile
else
	@echo "WARNING: uv not found, not compiling locales"
endif

# Check translations
check:
ifneq ($(UV),)
	uv run messages.py check
else
	@echo "ERROR: uv not found"
	@false
endif

# Remove .mo files
clean:
	rm -f translations/*/messages.mo
