# halmidi.spec
%global debug_package %{nil}

Name:           %{name}
Version:        %{version}
Release:        1%{?dist}
Summary:        S3 Object Server
BuildArch:      %{arch}
BuildRoot:      %{_tmppath}/%{name}-%{version}-build

License:        GPL
Source0:        %{name}-%{version}.tar.gz

%description
S3 Object Server

%prep
%setup -q

%build
# No build step needed, already built outside

%install
mkdir -p %{buildroot}/usr/bin
install -m 0755 %{name} %{buildroot}/usr/bin/%{name}

mkdir -p %{buildroot}/usr/lib/systemd/system
install -m 0644 %{name}.service %{buildroot}/usr/lib/systemd/system/%{name}.service

%files
/usr/bin/%{name}
/usr/lib/systemd/system/%{name}.service

%pre
getent passwd halmidiuser >/dev/null || useradd -r -s /sbin/nologin halmidiuser
if ! id halmidiuser &>/dev/null; then
    if ! useradd -r -s /sbin/nologin halmidiuser &>/dev/null; then
        echo "Failed to create user."
        exit 1
    fi
fi

%post
%systemd_post %{name}.service

chown halmidiuser /usr/bin/%{name}
chmod 700 /usr/bin/%{name}
systemctl daemon-reexec
systemctl daemon-reload
systemctl enable --now %{name}.service

%preun
%systemd_preun %{name}.service

%postun
%systemd_postun_with_restart %{name}.service

%changelog
* Sun Oct 05 2025 root
- Initial RPM package
