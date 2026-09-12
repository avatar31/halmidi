#!/bin/bash
set -e

SUCCESS=0
FAIL=1
if [[ $# -eq 0 ]]; then
        echo "Usage: $0 <build number>"
        echo "example: $0 0.9.8.12"
        exit ${FAIL}
fi

APP_NAME="halmidi"
VERSION=$1
ARCH="amd64"
WORKING_DIR=$PWD
BUILD_DIR="${APP_NAME}-deb/BUILD"
PACKAGE_NAME="${APP_NAME}_${VERSION}_${ARCH}.deb"
PACKAGE_ROOT="${APP_NAME}-deb/DEBS"

echo "🧹 Cleaning up old build..."
rm -rf "${APP_NAME}-deb"
mkdir -p \
  "$BUILD_DIR/DEBIAN" \
  "$BUILD_DIR/usr/bin" \
  "$BUILD_DIR/etc/systemd/system" \
  "$PACKAGE_ROOT"
cp postinst "$BUILD_DIR/DEBIAN/postinst"
cp postrm "$BUILD_DIR/DEBIAN/postrm"
cp preinst "$BUILD_DIR/DEBIAN/preinst"

chmod 755 "$BUILD_DIR/DEBIAN/postinst"
chmod 755 "$BUILD_DIR/DEBIAN/postrm"
chmod 755 "$BUILD_DIR/DEBIAN/preinst"

cd ..
./build_binary.sh
cd "${WORKING_DIR}"

echo "📝 Creating control file..."
cat <<EOF > "$BUILD_DIR/DEBIAN/control"
Package: $APP_NAME
Version: $VERSION
Section: base
Priority: optional
Architecture: $ARCH
Maintainer: Sachin S <sachin-s@hpe.com>
Description: $APP_NAME server
EOF

echo "📝 Creating systemd service..."
cp "../bin/${APP_NAME}" "$BUILD_DIR/usr/bin/$APP_NAME"
cp "../$APP_NAME.service" "$BUILD_DIR/etc/systemd/system/$APP_NAME.service"

echo "🔒 Setting permissions..."
chmod 755 "$BUILD_DIR/usr/bin/$APP_NAME"
chmod 644 "$BUILD_DIR/etc/systemd/system/$APP_NAME.service"
chmod 755 "$BUILD_DIR/DEBIAN"

echo "📦 Building .deb package..."
dpkg-deb --build "$BUILD_DIR" "$PACKAGE_ROOT/$PACKAGE_NAME"

echo "✅ Done: $PACKAGE_NAME"
