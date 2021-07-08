#!/bin/sh

# base functions.
. /etc/rc.d/init.d/functions

# app informations.
APPBIN="monstache"
APPARGS="-f ./etc/config.toml"
BINPATH="."

# start app.
start() {
    # start daemon.
    echo -n $"Starting ${APPBIN}: "
    daemon "${BINPATH}/${APPBIN} ${APPARGS} &"
    RET=$?
    echo
    return $RET
}

# stop app.
stop() {
    # stop daemon.
    echo -n $"Stopping ${APPBIN}: "
    killproc ${APPBIN}
    RET=$?
    echo
    return $RET
}

# restart app.
restart() {
    # stop app.
    stop

    # start app again.
    start
}

# monitor app.
monitor() {
    echo -n $"Monitor ${APPBIN}: "
    if [ -n "`pidofproc ${APPBIN}`" ] ; then
        success $"Monitor ${APPBIN}"
        echo
    else
        warning $"Monitor ${APPBIN} isn't running, restart it now..."
        echo
        start
    fi
}

# show daemon status.
status() {
    ps -ef | grep -w "${APPBIN}" | grep -v grep | grep -v -w sh
}

# switch cmd.
case "$1" in
    start)
        status && exit 0
        $1
    ;;
    stop)
        status || exit 0
        $1
    ;;
    restart)
        $1
    ;;
    status)
        $1
    ;;
    monitor)
        $1
    ;;
    *)
        echo $"Usage: $0 {start|stop|status|restart|monitor}"
        exit 2
esac
