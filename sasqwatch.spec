%define __spec_install_post %{nil}
%define debug_package %{nil}

Summary:   A modern take on the classic watch command for Linux
Name:      sasqwatch
Version:   0.2.5
Release:   4
License:   MIT
URL:       https://github.com/fabio42/sasqwatch
Source0:   https://github.com/fabio42/sasqwatch/archive/refs/tags/v%{version}.tar.gz

Requires:  procps-ng
Requires:  upx

%description
Sasqwatch is a modern take on the classic watch command for Linux. It periodically executes a command and displays the output in a clear and concise manner.

%prep
%setup -q

%build
GO111MODULE=on CGO_ENABLED=0 go build -ldflags="-s -w -X 'github.com/fabio42/sasqwatch/cmd.Version=%{version}'" -o %{name}
strip %{name}
upx %{name}

%install
%{__install} -Dm755 %{name} %{buildroot}%{_bindir}/%{name}

%files
%{_bindir}/%{name}

%changelog
* Mon Jun 10 2024 Danie de Jager <danie.dejager@gmail.com> - 0.2.5-4
* Fri Jul 28 2023 Danie de Jager <danie.dejager@gmail.com> - 0.2.5-2
- Improved printing of version information
* Tue Jun 20 2023 Danie de Jager <danie.dejager@gmail.com> - 0.2.5-1
- Initial package release
