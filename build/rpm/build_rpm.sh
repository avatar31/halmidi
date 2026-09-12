#!/bin/bash
set -e
# set -x

SUCCESS=0
FAIL=1
if [[ $# -eq 0 ]]; then
        echo "Usage: $0 <build number>"
        echo "example: $0 0.9.8.12"
        exit ${FAIL}
fi

APP_NAME="halmidi"
VERSION=$1
ARCH="x86_64"
WORKING_DIR=$PWD
TOPDIR="$WORKING_DIR/rpmbuild"
PACKAGE_NAME="${APP_NAME}_${VERSION}_${ARCH}.rpm"

echo "🧹 Cleaning up old build..."
rm -rf "${TOPDIR}"

mkdir -p \
    $TOPDIR/SPECS \
    $TOPDIR/BUILD \
    $TOPDIR/RPMS \
    $TOPDIR/SOURCES \
    $TOPDIR/SRPMS

cd ..
./build_binary.sh
cd "${WORKING_DIR}"

echo "📦 Copying binary to SOURCES..."
mkdir -p "${TOPDIR}/SOURCES/${APP_NAME}-${VERSION}"
cp "../bin/${APP_NAME}" "${TOPDIR}/SOURCES/${APP_NAME}-${VERSION}/"
cp "../${APP_NAME}.service" "${TOPDIR}/SOURCES/${APP_NAME}-${VERSION}/"

cd "$TOPDIR/SOURCES"
tar -czvf "${APP_NAME}-${VERSION}.tar.gz" "${APP_NAME}-${VERSION}"

cd "${WORKING_DIR}"
cp "${APP_NAME}.spec" "${TOPDIR}/SPECS/"

echo "📦 Building .rpm package..."
rpmbuild --define "_topdir $TOPDIR" -bb ${TOPDIR}/SPECS/${APP_NAME}.spec -D "version $VERSION" -D "name $APP_NAME" -D "arch $ARCH"

echo "✅ Done: $TOPDIR/RPMS/$ARCH/$PACKAGE_NAME"
