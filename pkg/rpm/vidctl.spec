Name:           vidctl
Version:        1.0.1
Release:        1%{?dist}
Summary:        Compress video to the exact size of each platform with ffmpeg 2-pass

License:        MIT
URL:            https://github.com/fabianoflorentino/vidctl
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  gcc
BuildRequires:  golang
BuildRequires:  nodejs
BuildRequires:  pkgconf-pkg-config
BuildRequires:  gtk3-devel
BuildRequires:  webkit2gtk4.1-devel

Requires:       gtk3
Requires:       webkit2gtk4.1
# ffmpeg fica como recomendação: no Fedora oficial ele vive no RPM Fusion.
Recommends:     ffmpeg

%description
vidctl compresses videos to the exact size of each platform using ffmpeg
two-pass encode, keeping duration and format intact. Desktop app built with
Wails and Go, 100% offline.

%prep
%setup -q -n vidctl-%{version}

%build
export PATH="$PATH:$(go env GOPATH)/bin"
go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0
wails build -clean -tags webkit2_41

%install
install -Dm755 build/bin/vidctl %{buildroot}%{_bindir}/vidctl
install -Dm644 pkg/vidctl.desktop %{buildroot}%{_datadir}/applications/vidctl.desktop
install -Dm644 pkg/icon/vidctl-128.png %{buildroot}%{_datadir}/icons/hicolor/128x128/apps/vidctl.png
install -Dm644 pkg/icon/vidctl-256.png %{buildroot}%{_datadir}/icons/hicolor/256x256/apps/vidctl.png

%files
%{_bindir}/vidctl
%{_datadir}/applications/vidctl.desktop
%{_datadir}/icons/hicolor/128x128/apps/vidctl.png
%{_datadir}/icons/hicolor/256x256/apps/vidctl.png

%changelog
* Fri Sep 11 2026 Fabiano Santos Florentino <fabianoflorentino@gmail.com> - 1.0.1-1
- Empacotamento inicial (Fedora)