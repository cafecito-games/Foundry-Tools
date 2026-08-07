namespace cafecito.json.v1

## Cases of the choice protobuf oneof.
enum_name JsonSuiteChoiceCase:
	Note(note: String)
	Tally(tally: long)
	Detail(detail: Reference)
