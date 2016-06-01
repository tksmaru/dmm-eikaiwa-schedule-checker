# -*- coding: utf-8 -*-

# https://github.com/prmtl/appfy.recipe.gae/blob/master/appfy/recipe/gae/sdk.py より一部改変

from distutils import version
import urllib2
import json
import re

class HeadRequest(urllib2.Request):
    def get_method(self):
        return "HEAD"

GO_SDK_RE = re.compile(r'featured/go_appengine_sdk_linux_amd64-(\d+\.\d+\.\d+).zip')
URL = "https://www.googleapis.com/storage/v1/b/appengine-sdks/o?prefix=featured"

raw_bucket_list = urllib2.urlopen(URL).read()
bucket_list = json.loads(raw_bucket_list)

all_sdks = bucket_list['items']
go_sdks = [
    sdk for sdk in all_sdks if GO_SDK_RE.match(sdk['name'])
]

# 1.9.38 > 1.9.37 so we need reverse order
def version_key(sdk):
    version_string = GO_SDK_RE.match(sdk['name']).group(1)
    return version.StrictVersion(version_string)

go_sdks.sort(key=version_key, reverse=True)


# Newest listed versions are not immediately available to download.
# Check over HEAD.
for sdk in go_sdks:
    url = str(sdk['mediaLink'])
    try:
        request = HeadRequest(url)
        urllib2.urlopen(request)
    except urllib2.HTTPError as e:
        # print "not exists or not yet published"
        continue
    else:
        print url
        break
