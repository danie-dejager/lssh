Name:           lssh
Version:        0.6.8
Release:        1%{?dist}
Summary:        List selection type ssh/scp/sftp client command. Supports single connection and parallel connection. Local bashrc can also be used without placing it on a remote machine. Written in Golang.
URL:            https://github.com/blacknon/lssh/
License:        MIT
Source0:        https://github.com/blacknon/lssh/archive/refs/tags/v%{version}.tar.gz
Source1:        https://raw.githubusercontent.com/blacknon/lssh/master/example/config.tml

BuildRequires:  git
BuildRequires:  python3
BuildRequires:  curl
BuildRequires:  gcc
BuildRequires:  golang
BuildRequires:  systemd-rpm-macros

%define debug_package %{nil}

%description
This command utility to read a prepared list in advance and connect ssh/scp/sftp the selected host. List file is set in yaml format. When selecting a host, you can filter by keywords. Can execute commands concurrently to multiple hosts.

lsftp shells can be connected in parallel.

Supported multiple ssh proxy, http/socks5 proxy, x11 forward, and port forwarding.

%prep
%autosetup -n %{name}-%{version}

%build
GO111MODULE=auto go build -o lssh  github.com/blacknon/lssh/cmd/lssh
GO111MODULE=auto go build -o lscp  github.com/blacknon/lssh/cmd/lscp
GO111MODULE=auto go build -o lsftp github.com/blacknon/lssh/cmd/lsftp

%install
# Install the binary
install -D -m 0755 %{name} %{buildroot}/usr/bin/%{name}
install -D -m 0755 lscp %{buildroot}/usr/bin/%{name}
install -D -m 0755 lsftp %{buildroot}/usr/bin/%{name}

# Download the external source file if it doesn't exist and install it
%{__mkdir_p} %{buildroot}/etc/skel
install %{SOURCE1} %{buildroot}/etc/skel/lssh.conf

%files
%license LICENSE.md
%doc README.md
/usr/bin/%{name}
/etc/skel/lssh.conf

%changelog
* Mon Jul 8 2024 Danie de Jager - 0.6.8-1
 - Initial version
