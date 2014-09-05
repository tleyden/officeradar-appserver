
This is the app server for [office-radar](https://github.com/tleyden/office-radar), which is an example app built on top of [Couchbase Lite](https://github.com/couchbase/couchbase-lite-ios).

Here's the overall architecture for OfficeRadar.  This repository contains the code that corresponds to the yellow box.

![architecture diagram](http://tleyden-misc.s3.amazonaws.com/blog_images/officeradar_appserver_architecture.png)

What this code does:

* Listens to the [Sync Gateway](https://github.com/couchbase/sync_gateway) `_changes` feed for new changes
* If the new changes meet certain criteria, push notifications are sent out to Apple's Push Notification service, via [Uniqush](http://uniqush.org/) 