TODO: unit test organizer object, implement transition from finished test to finished testing.
additionally, implement transition from each event to internal server error, and from finished compilation to compilation error.
i think it would be much prettier to:
- remove started evaluation event
- remove finished evaluation event
- add compilation error event
- add internal server error event
that would certainly make the handling easier.