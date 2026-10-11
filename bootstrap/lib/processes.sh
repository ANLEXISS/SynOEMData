#!/bin/sh
scan_processes() {
    PID1_KIND=unknown; SERVICE_MANAGER=unknown; PROCESS_SUPERVISOR=unknown
    _comm=$(read_one /proc/1/comm); _exe=$(readlink "$(path /proc/1/exe)" 2>/dev/null || :); _base=${_exe##*/}
    case $_comm in systemd) PID1_KIND=systemd;; procd) PID1_KIND=procd;; busybox) PID1_KIND=busybox_init;; tini|docker-init) PID1_KIND=external;; s6-svscan) PID1_KIND=s6;; runsvdir) PID1_KIND=runit;; init) [ -r "$(path /etc/inittab)" ] && PID1_KIND=sysvinit;; esac
    if [ "$PID1_KIND" = unknown ]; then case $_base in systemd) PID1_KIND=systemd;; procd) PID1_KIND=procd;; busybox) PID1_KIND=busybox_init;; tini|docker-init) PID1_KIND=external;; s6-svscan) PID1_KIND=s6;; runsvdir) PID1_KIND=runit;; esac; fi
    case $PID1_KIND in systemd) [ -d "$(path /run/systemd/system)" ] && SERVICE_MANAGER=systemd;; procd) SERVICE_MANAGER=procd;; busybox_init) SERVICE_MANAGER=busybox_init;; sysvinit) SERVICE_MANAGER=sysvinit;; s6) SERVICE_MANAGER=s6;; runit) SERVICE_MANAGER=runit;; external) SERVICE_MANAGER=external;; esac
    if [ -d "$(path /run/openrc)" ] && [ "$SERVICE_MANAGER" != systemd ] && [ "$SERVICE_MANAGER" != procd ] && [ "$SERVICE_MANAGER" != external ]; then SERVICE_MANAGER=openrc; fi
    if [ "$SERVICE_MANAGER" = unknown ] && [ -e "$(path /etc/rc.common)" ]; then SERVICE_MANAGER=procd; fi
    [ "$SERVICE_MANAGER" = unknown ] && [ -d "$(path /etc/init.d)" ] && [ -r "$(path /etc/inittab)" ] && SERVICE_MANAGER=sysvinit
    case $SERVICE_MANAGER in systemd|procd|openrc|busybox_init|sysvinit|runit|s6|upstart) PROCESS_SUPERVISOR=$SERVICE_MANAGER;; external) PROCESS_SUPERVISOR=external;; esac
}
processes_json() { printf '{"pid1":'; status_value_json "$( [ "$PID1_KIND" = unknown ] && echo unknown || echo detected )" "$PID1_KIND" '/proc/1/comm,/proc/1/exe,/proc/1/cmdline'; printf ',"service_manager":'; json_quote "$SERVICE_MANAGER"; printf ',"process_supervisor":'; json_quote "$PROCESS_SUPERVISOR"; printf ',"container_external_supervision":%s}' "$( [ "$SERVICE_MANAGER" = external ] && echo true || echo false )"; }
