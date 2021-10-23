package errorinfo

const (
	//event error 001
	GET_EVENT_FROM_DB_ERROR_CODE = "500001001"
	GET_EVENT_FROM_DB_ERROR_MSG  = "Get event data from db error"

	//http request
	HTTP_REQUEST_PARAMS_JSON_FORMAT_ERROR_CODE        = "500002001"
	HTTP_REQUEST_PARAMS_JSON_FORMAT_ERROR_MSG         = "Request JSON format was error"
	HTTP_REQUEST_PARAMS_NULL_ERROR_CODE               = "500002002"
	HTTP_REQUEST_PARAMS_NULL_ERROR_MSG                = "Request params value can not be null"
	PAGE_NUMBER_OR_SIZE_FORMAT_ERROR_CODE             = "500002003"
	PAGE_NUMBER_OR_SIZE_FORMAT_ERROR_MSG              = "Page number or page size format was wrong"
	HTTP_REQUEST_SEND_REQUEST_RETUREN_ERROR_CODE      = "500002004"
	HTTP_REQUEST_SEND_REQUEST_RETUREN_ERROR_MSG       = "return error when sending http request"
	HTTP_REQUEST_PARSER_RESPONSE_TO_STRUCT_ERROR_CODE = "500002005"
	HTTP_REQUEST_PARSER_RESPONSE_TO_STRUCT_ERROR_MSG  = "Parse http request response to struct occurred error"

	//database err 003
	GET_RECORD_COUNT_ERROR_CODE = "500003001"
	GET_RECORD_COUNT_ERROR_MSG  = "Get data total count from db occurred error"
	GET_RECORD_lIST_ERROR_CODE  = "500003002"
	GET_RECORD_lIST_ERROR_MSG   = "Get data records from db occurred error"

	// JWT token err 004
	NO_AUTHORIZATION_TOKEN_ERROR_CODE = "500004001"
	NO_AUTHORIZATION_TOKEN_ERROR_MSG  = "Jwt token is required"

	//lotus send deal err  005
	SENDING_DEAL_ERROR_CODE                 = "500005001"
	SENDING_DEAL_ERROR_MSG                  = "Failed to send deal"
	GET_LATEST_PRICE_OF_FILECOIN_ERROR_CODE = "500005002"
	GET_LATEST_PRICE_OF_FILECOIN_ERROR_MSG  = "Getting filecion latest price occurred error"
	SENDING_DEAL_GET_NULL_RETURN_VALUE_CODE = "500005003"
	SENDING_DEAL_GET_NULL_RETURN_VALUE_MSG  = "Get null return value when deal had been sent"

	//system command err  006
	GET_HOME_DIR_ERROR_CODE = "500006001"
	GET_HOME_DIR_ERROR_MSG  = "Getting home dir occurred error"

	CREATE_DIR_ERROR_CODE = "500006002"
	CREATE_DIR_ERROR_MSG  = "Creating dir occurred error"
)