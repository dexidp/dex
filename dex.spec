Name: dex
Version: v2.11.0
Release: 1%{?dist}
License: ASL 2.0
Summary: OpenID Connect Identity (OIDC) Provider

URL: https://github.com/dexidp/dex
Source0: %{name}-%{version}.tar.gz

%description
OpenID Connect Identity (OIDC) and OAuth 2.0 Provider with Pluggable Connectors

%pre
# Add the "dex" user
/usr/sbin/useradd -c "Dex" \
	-s /sbin/nologin -r -d %{_datarootdir}/dex/ dex 2> /dev/null || :

%post
if [ $1 -eq 1 ] ; then
        # Initial installation
        systemctl preset dex.service >/dev/null 2>&1 || :
fi

%preun
if [ $1 -eq 0 ] ; then
        # Package removal, not upgrade
        systemctl --no-reload disable dex.service > /dev/null 2>&1 || :
        systemctl stop dex.service > /dev/null 2>&1 || :
fi

%postun
systemctl daemon-reload >/dev/null 2>&1 || :

%prep
%setup -q

%build
export LDFLAGS=-linkmode=external
mkdir -p ./_build/src/github.com/dexidp
ln -s $(pwd) ./_build/src/github.com/dexidp/dex
export GOPATH=$(pwd)/_build:%{gopath}
export VERSION=%{version}
make bin/dex
make bin/grpc-client

%install
install -m 755 -d %{buildroot}%{_sysconfdir}/dex/
install -m 755 -d %{buildroot}%{_bindir}
install -m -o dex -g dex 750 -d %{buildroot}%{_localstatedir}/dex/
install -m 755 -d %{buildroot}%{_datarootdir}/dex/
install -m 755 -d %{buildroot}%{_sysconfdir}/systemd/system/
install -m 755 -d %{buildroot}%{_sysconfdir}/systemd/system
install -p -m 755 -t %{buildroot}%{_bindir} bin/dex
install -p -m 755 -t %{buildroot}%{_bindir} bin/grpc-client
install -p -m 644 -t %{buildroot}%{_sysconfdir}/systemd/system rpm/dex.service
install -p -m -o dex -g dex 640 -t %{buildroot}%{_sysconfdir}/dex rpm/config.yaml
cp -a web %{buildroot}%{_datarootdir}/dex

%files
%{_bindir}/dex
%{_bindir}/grpc-client
%{_datarootdir}/dex/*
%{_sysconfdir}/dex rpm/config.yaml
%{_sysconfdir}/systemd/system/dex.service
%{_localstatedir}/dex
