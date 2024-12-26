TODO: if we can get evaluation from another service object, it means that the
s3 repository is WORKING as expected. that should be included in the test.

Next start integrating with submission service. I think that at least for now
the postgres could store a simple array with scoring information.

We noticed that if the memory limit is too low, segmentation fault is received.
We could for each programming language specify the minimum memory limit.