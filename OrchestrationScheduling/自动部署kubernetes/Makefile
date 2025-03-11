PKG_NAME = cncos-ops
VER = v0.0.2
CURRENT_VERSION = release-$(VER)

clean:
	-rm ./cncos-ops-script-release-$(VER)*.tar.gz
	-rm ./cncos-ops-offline-release-$(VER)-k8s-*.tgz
	-rm -r ./cncos-ops-offline
	-rm MD5SUMS

build:clean
	find . -not -path "*/.git/*" -a -not -path "*/cncos-ops-offline/*" -a -not -path "*/charts/*" -a -not -path "*/Makefile" -a -not -path "*/functions/*" -a -not -path "*.yaml" -a -not -path "*.md" -type f -print0 | xargs -0 chmod 555
	find ./functions/ -not -path "*/.git/*" -a -not -path "*/cncos-ops-offline/*" -a -not -path "*/charts/*" -a -not -path "*/Makefile"  -type f -print0 | xargs -0 chmod 444
	tar -czvf cncos-ops-script-release-$(VER).tar.gz --exclude=cncos-ops-offline --exclude=Makefile --exclude=\..* --exclude=.*tar.gz ./*
	md5sum cncos-ops-script-release-$(VER).tar.gz >> MD5SUMS

build_offline_pkg:clean
	find . -not -path "*/.git/*" -a -not -path "*/cncos-ops-offline/*" -a -not -path "*/charts/*" -a -not -path "*/Makefile" -a -not -path "*/functions/*" -a -not -path "*.yaml" -a -not -path "*.md" -type f -print0 | xargs -0 chmod 555
	find ./functions/ -not -path "*/.git/*" -a -not -path "*/cncos-ops-offline/*" -a -not -path "*/charts/*" -a -not -path "*/Makefile" -type f -print0 | xargs -0 chmod 444
	tar -czvf cncos-ops-script-release-$(VER).tar.gz --exclude=cncos-ops-offline --exclude=Makefile --exclude=\..* --exclude=.*tar.gz ./*
	md5sum cncos-ops-script-release-$(VER).tar.gz >> MD5SUMS
	./offline_package.sh env/offline-manifest.yaml
	for f in $(find cncos-ops-offline -iname cncos-ops-offline-*.tgz -type f);do
		mv $f cncos-ops-offline-release-$(VER)-k8s-${f##*-}
	done
	md5sum cncos-ops-offline-release-$(VER)-k8s-*.tgz >> MD5SUMS
