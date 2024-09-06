#!/bin/bash
# Copyright © 2024 Alexandre Pires

# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:

# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.

# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.

CONFIGFILE=${CONFIGFILE:-"conf/m3uproxy.json"}
USERNAME=${USERNAME:-"admin"}
PASSWORD=${PASSWORD:-"admin"}

# Check if users.json exists, if not add users from environment variables
if [ ! -f $USERSFILE ]; then
  echo "Adding initial user."
  if [ -n "$USERNAME" ] && [ -n "$PASSWORD" ]; then
    echo "Adding user $USERNAME"
    /app/m3uproxy users add -c $CONFIGFILE $USERNAME $PASSWORD
    if [ $? -ne 0 ]; then
        echo "Failed to add user $USERNAME"
        exit 1
    fi
  fi
fi

# Start m3uproxy
/app/m3uproxy server -c $CONFIGFILE