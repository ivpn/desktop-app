
#include <stdio.h>
#include <string.h>
#include <stdarg.h>

#include <oqs/oqs.h>

#include "jsmn.h"
#include "base64.h"

#define APP_VERSION "1.1.0"

#define SUCCESS 0
#define ERROR	-1

void fprintf_stderr(const char *format, ...) {
    va_list args;
    fputs("{\"error\":\"", stderr);

    va_start(args, format);
    vfprintf(stderr, format, args);
    va_end(args);

    fputs("\"}\n", stderr);
}

void* secure_realloc(void* ptr, size_t old_size, size_t new_size) {
	if (new_size == 0) {
		OQS_MEM_secure_free(ptr, old_size);
		return NULL;
	}

	if (ptr == NULL) return malloc(new_size);

	void* new_ptr = malloc(new_size);
	if (new_ptr == NULL) return NULL;

	size_t size_to_copy = (old_size < new_size) ? old_size : new_size;
	memcpy(new_ptr, ptr, size_to_copy);
	OQS_MEM_secure_free(ptr, old_size);

	return new_ptr;
}

char* read_data_from_stdin(const size_t initial_buffer, const size_t max_buff_size, size_t* data_len) {
	size_t data_length = 0;
	size_t buffer_size = initial_buffer;
	if (buffer_size == 0)
		buffer_size = 1024 * 10;

	char* buffer = (char*)malloc(buffer_size + 1);  // Allocate an extra byte for the null terminator

	if (buffer == NULL) {
		fprintf_stderr("Memory allocation failed.");
		return NULL;
	}

	while (1) {
		size_t bytes_read = fread(buffer + data_length, 1, buffer_size - data_length, stdin);

		data_length += bytes_read;

		if (bytes_read < buffer_size - data_length || feof(stdin)) {
			if (feof(stdin)) {
				break;
			}
			else if (ferror(stdin)) {
				fprintf_stderr("Reading from stdin failed.");
				OQS_MEM_secure_free(buffer, buffer_size + 1);
				return NULL;
			}
		}

		if (data_length >= buffer_size) {
			size_t buffer_size_old = buffer_size + 1; // extra byte allocated for the null terminator	
			buffer_size *= 2;

			if (max_buff_size) {
				if (buffer_size_old >= max_buff_size) {
					fprintf_stderr("Too much data (max buffer size=%zd).", max_buff_size);
					OQS_MEM_secure_free(buffer, buffer_size + 1);
					return NULL;
				}
				if (buffer_size + 1 > max_buff_size) {  // extra byte allocated for the null terminator	
					buffer_size = max_buff_size - 1;
				}
			}

			buffer = (char*)secure_realloc(buffer, buffer_size_old, buffer_size + 1);  // Allocate an extra byte for the null terminator

			if (buffer == NULL) {
				fprintf_stderr("Memory allocation failed.");
				return NULL;
			}
		}
	}

	// Null-terminate the string
	buffer[data_length] = '\0';	
	// Resize the buffer to the actual size of the data 
	// (skipping this step to increase speed)
	// buffer = (char*)secure_realloc(buffer, buffer_size + 1, data_length + 1);

	if (data_len) *data_len = data_length;
	return buffer;
}

void remove_newlines(unsigned char* str) {
	unsigned char* r = str, * w = str;
	while (*r != '\0') {
		if (*r != '\n' && *r != '\r') {
			*w = *r;
			w++;
		}
		r++;
	}
	*w = '\0';
}

void cleanup(uint8_t* secret_key, uint8_t* shared_secret, uint8_t* public_key, uint8_t* ciphertext, OQS_KEM* kem) {
	if (kem != NULL) {
		OQS_MEM_secure_free(secret_key, kem->length_secret_key);
		OQS_MEM_secure_free(shared_secret, kem->length_shared_secret);
	}
	OQS_MEM_insecure_free(public_key);
	OQS_MEM_insecure_free(ciphertext);
	OQS_KEM_free(kem);
}

int generate_keys(FILE* const _out_stream, const char* kem_alg_name) {
	OQS_KEM* kem = NULL;
	uint8_t* public_key = NULL;
	uint8_t* secret_key = NULL;

	kem = OQS_KEM_new(kem_alg_name);
	if (kem == NULL) {
		fprintf_stderr("%s was not enabled at compile-time.", kem_alg_name);
		return ERROR;
	}

	public_key = malloc(kem->length_public_key);
	secret_key = malloc(kem->length_secret_key);
	if (!public_key || !secret_key) {
		fprintf_stderr("malloc failed!");
		cleanup(secret_key, NULL, public_key, NULL, kem);
		return ERROR;
	}

	OQS_STATUS rc = OQS_KEM_keypair(kem, public_key, secret_key);
	if (rc != OQS_SUCCESS) {
		fprintf_stderr("OQS_KEM_keypair failed!");
		cleanup(secret_key, NULL, public_key, NULL, kem);
		return ERROR;
	}

	size_t sec_key_base_64_len = 0;
	size_t pub_key_base64_len = 0;
	unsigned char* sec_key_base64 = base64_encode(secret_key, kem->length_secret_key, &sec_key_base_64_len);
	unsigned char* pub_key_base64 = base64_encode(public_key, kem->length_public_key, &pub_key_base64_len);
	if (!sec_key_base64 || !pub_key_base64) {
		fprintf_stderr("base64_encode() failed!");
		OQS_MEM_secure_free(sec_key_base64, sec_key_base_64_len);
		OQS_MEM_secure_free(pub_key_base64, pub_key_base64_len);
		cleanup(secret_key, NULL, public_key, NULL, kem);
		return ERROR;
	}

	remove_newlines(sec_key_base64);
	remove_newlines(pub_key_base64);
	fprintf(_out_stream, "{\n\"priv\":\"%s\",\n\"pub\":\"%s\",\n\"lib_ver\":\"%s\"\n}\n", sec_key_base64, pub_key_base64, OQS_VERSION_TEXT);

	OQS_MEM_secure_free(sec_key_base64, sec_key_base_64_len);
	OQS_MEM_secure_free(pub_key_base64, pub_key_base64_len);
	cleanup(secret_key, NULL, public_key, NULL, kem);

	return SUCCESS;
}

int encode_preshared_key(FILE* const _out_stream, const char* kem_alg_name, const char* public_key_base64) {
	OQS_KEM* kem = NULL;
	uint8_t* public_key = NULL;
	uint8_t* ciphertext = NULL;
	uint8_t* shared_secret = NULL;

	size_t	 public_key_len = 0;
	public_key = base64_decode((const unsigned char*) public_key_base64, strlen(public_key_base64), &public_key_len);
	if (!public_key) {
		fprintf_stderr("base64_decode() failed!");
		return ERROR;
	}

	kem = OQS_KEM_new(kem_alg_name);
	if (kem == NULL) {
		fprintf_stderr("%s was not enabled at compile-time.", kem_alg_name);
		cleanup(NULL, shared_secret, public_key, ciphertext, kem);
		return ERROR;
	}

	if (public_key_len != kem->length_public_key) {
		fprintf_stderr("unexpected length of public key for %s!", kem_alg_name);
		cleanup(NULL, shared_secret, public_key, ciphertext, kem);
		return ERROR;
	}

	ciphertext = malloc(kem->length_ciphertext);
	shared_secret = malloc(kem->length_shared_secret);
	if (!ciphertext || !shared_secret) {
		fprintf_stderr("malloc failed!");
		cleanup(NULL, shared_secret, public_key, ciphertext, kem);
		return ERROR;
	}

	int rc = OQS_KEM_encaps(kem, ciphertext, shared_secret, public_key);
	if (rc != OQS_SUCCESS) {
		fprintf_stderr("OQS_KEM_encaps failed!");
		cleanup(NULL, shared_secret, public_key, ciphertext, kem);
		return ERROR;
	}

	size_t ciphertext_base_64_len = 0;
	size_t shared_secret_base64_len = 0;
	unsigned char* ciphertext_base64 = base64_encode(ciphertext, kem->length_ciphertext, &ciphertext_base_64_len);
	unsigned char* shared_secret_base64 = base64_encode(shared_secret, kem->length_shared_secret, &shared_secret_base64_len);
	if (!ciphertext_base64 || !shared_secret_base64) {
		fprintf_stderr("base64_encode() failed!");
		OQS_MEM_secure_free(ciphertext_base64, ciphertext_base_64_len);
		OQS_MEM_secure_free(shared_secret_base64, shared_secret_base64_len);
		cleanup(NULL, shared_secret, public_key, ciphertext, kem);
		return ERROR;
	}

	remove_newlines(shared_secret_base64);
	remove_newlines(ciphertext_base64);
	fprintf(_out_stream, "{\n\"secret\":\"%s\",\n\"cipher\":\"%s\"\n}\n", shared_secret_base64, ciphertext_base64);


	OQS_MEM_secure_free(ciphertext_base64, ciphertext_base_64_len);
	OQS_MEM_secure_free(shared_secret_base64, shared_secret_base64_len);
	cleanup(NULL, shared_secret, public_key, ciphertext, kem);
	return SUCCESS;
}

int decode_preshared_key(FILE* const _out_stream, const char* kem_alg_name, const char* secret_key_base64, const char* ciphertext_base64) {
	OQS_KEM* kem = NULL;
	uint8_t* ciphertext = NULL;
	uint8_t* shared_secret = NULL;
	uint8_t* secret_key = NULL;

	size_t	 secret_key_len = 0;
	size_t	 ciphertext_len = 0;
	secret_key = base64_decode((const unsigned char*) secret_key_base64, strlen(secret_key_base64), &secret_key_len);
	ciphertext = base64_decode((const unsigned char*) ciphertext_base64, strlen(ciphertext_base64), &ciphertext_len);
	if (!secret_key || !ciphertext) {
		OQS_MEM_secure_free(secret_key, secret_key_len);
		OQS_MEM_secure_free(ciphertext, ciphertext_len);
		fprintf_stderr("base64_decode() failed!");
		return ERROR;
	}

	kem = OQS_KEM_new(kem_alg_name);
	if (kem == NULL) {
		fprintf_stderr("%s was not enabled at compile-time.", kem_alg_name);
		cleanup(secret_key, shared_secret, NULL, ciphertext, kem);
		return ERROR;
	}

	if (secret_key_len != kem->length_secret_key) {
		fprintf_stderr("unexpected length of secret key for %s!", kem_alg_name);
		OQS_MEM_secure_free(secret_key, secret_key_len);
		cleanup(NULL, shared_secret, NULL, ciphertext, kem);
		return ERROR;
	}

	shared_secret = malloc(kem->length_shared_secret);
	if (!shared_secret) {
		fprintf_stderr("malloc failed!");
		cleanup(secret_key, shared_secret, NULL, ciphertext, kem);
		return ERROR;
	}

	int rc = OQS_KEM_decaps(kem, shared_secret, ciphertext, secret_key);
	if (rc != OQS_SUCCESS) {
		fprintf_stderr("OQS_KEM_decaps failed!");
		cleanup(secret_key, shared_secret, NULL, ciphertext, kem);
		return ERROR;
	}

	size_t shared_secret_base64_len = 0;
	unsigned char* shared_secret_base64 = base64_encode(shared_secret, kem->length_shared_secret, &shared_secret_base64_len);
	if (!shared_secret_base64) {
		fprintf_stderr("base64_encode() failed!");
		OQS_MEM_secure_free(shared_secret_base64, shared_secret_base64_len);
		cleanup(secret_key, shared_secret, NULL, ciphertext, kem);
		return ERROR;
	}

	remove_newlines(shared_secret_base64);
	fprintf(_out_stream, "{\n\"secret\":\"%s\"\n}\n", shared_secret_base64);

	OQS_MEM_secure_free(shared_secret_base64, shared_secret_base64_len);
	cleanup(secret_key, shared_secret, NULL, ciphertext, kem);
	return SUCCESS;
}

int parse_json(jsmn_parser *p, jsmntok_t *t, char* json_string) {
	jsmn_init(p);
	int r = jsmn_parse(p, json_string, strlen(json_string), t, sizeof(*t) / sizeof(t[0]));
	if (r < 0) {
		fprintf_stderr("Failed to parse JSON: %d", r);
		return ERROR;
	}
	
	if (r < 1 || t[0].type != JSMN_OBJECT) { // Assume the top-level element is an object 
		fprintf_stderr("Failed to parse JSON: object expected");
		return ERROR;
	}
	return SUCCESS;
}

static int jsoneq(const char* json, jsmntok_t* tok, const char* s) {
	if (tok->type == JSMN_STRING && (int)strlen(s) == tok->end - tok->start &&
		strncmp(json + tok->start, s, tok->end - tok->start) == 0) {
		return 0;
	}
	return -1;
}
char* get_json_field(const char* name, jsmntok_t *t, int tokens, char* json_data) {
	for (int i = 1; i < tokens; i++) {
		if (jsoneq(json_data, &t[i], name) == 0) {
			json_data[t[i + 1].end] = 0;
			return &json_data[t[i + 1].start];
		}
	}
	return NULL;
}

int print_supported_kems(FILE* const _out_stream) {
	for (size_t i = 0; i < OQS_KEM_alg_count(); i++) {
		const char* kem = OQS_KEM_alg_identifier(i);
		if (OQS_KEM_alg_is_enabled(kem))
			fprintf(_out_stream, "  %s\n", kem);
	}
	return 0;
}

void print_version() {
	printf("KEM helper v%s; liboqs v%s (%s)\n", APP_VERSION, OQS_VERSION_TEXT, OQS_COMPILE_BUILD_TARGET);
}

void print_usage(const char* program_name) {
	printf("Usage:\n");
	printf("  %s version\n"
		"      -  print version info\n"
		, program_name);
	printf(	"  %s genkeys [<kem_algorithm_name>]\n"
			"      -  generate public and private keys\n"
			"      -  no input data required\n"
			"      -  output format:  '{\"priv\":\"...\", \"pub\":\"...\", \"lib_ver\":\"...\"}'\n"
		, program_name);
	printf("  %s encpsk  [<kem_algorithm_name>]>]\n"
			"      -  generate cipher text (encode PresharedKey using public key)\n"
			"      -  input format : '{\"pub\":\"...\"}'\n"
			"      -  output format: '{\"secret\":\"...\", \"cipher\":\"...\"}'\n"
		, program_name);
	printf("  %s decpsk  [<kem_algorithm_name>]\n"
			"      -  decode cipher text into PresharedKey (using private key)\n"
			"      -  input format : '{\"cipher\":\"...\", \"priv\":\"...\"}'\n"
			"      -  output format: '{\"secret\":\"...\"}'\n"
		, program_name);
	printf("  %s list_kems\n"
		"      -  list of supported KEM's (key encapsulation mechanisms)\n"
		, program_name);
}

int main(int argc, char* argv[]) {
	if (argc < 2) {
		print_version();
		print_usage("");
		return 1;
	}

	char* input_adata	   = NULL;
	size_t input_adata_len = 0;

	const int BUF_LEN_INIT = 1024 * 10;
	const int BUF_LEN_MAX  = 1024 * 1024 * 5;

	if (strcmp(argv[1], "version") == 0) {
		print_version();
		return 0;
	}

	if (strcmp(argv[1], "list_kems") == 0) {
		return print_supported_kems(stdout);
	}
		
	char* kem_algorithm_name = ""; 
	if (argc <= 2 || strlen(argv[2])==0) {
		fprintf_stderr("Input data error: kem_algorithm_name is not specified");			
		return ERROR;		
	}
	kem_algorithm_name = argv[2];

	if (strcmp(argv[1], "genkeys") == 0) {
		return generate_keys(stdout, kem_algorithm_name);
	}
	else if (strcmp(argv[1], "encpsk") == 0 || strcmp(argv[1], "decpsk") == 0) {
		// read JSON from stdin
		input_adata = read_data_from_stdin(BUF_LEN_INIT, BUF_LEN_MAX, &input_adata_len);
		if (input_adata == NULL) 
			return ERROR;			
			
		// Init & parse JSON
		int r;
		jsmn_parser p;
		jsmntok_t t[16]; // We expect no more than 16 tokens 
		jsmn_init(&p);
		r = jsmn_parse(&p, input_adata, strlen(input_adata), t,	sizeof(t) / sizeof(t[0]));
		if (r < 0) {
			fprintf_stderr("Failed to parse JSON: %d", r);
			return 1;
		}		
		if (r < 1 || t[0].type != JSMN_OBJECT) { // Assume the top-level element is an object 
			fprintf_stderr("Failed to parse JSON: object expected");
			return 1;
		}
			
		// process commands
		if (strcmp(argv[1], "encpsk") == 0) {

			const char* public_key = get_json_field("pub", t, r, input_adata);
			if (!public_key) {
				fprintf_stderr("required parameter not defined in JSON");
				print_usage("");
				return 1;
			}
			return encode_preshared_key(stdout, kem_algorithm_name, public_key);
		}
		else if (strcmp(argv[1], "decpsk") == 0) {
			const char* ciphertext = get_json_field("cipher", t, r, input_adata);
			const char* private_key = get_json_field("priv", t, r, input_adata);
			if (!ciphertext || !private_key) {
				fprintf_stderr("required parameter not defined in JSON");
				print_usage("");
				return 1;
			}
			return decode_preshared_key(stdout, kem_algorithm_name, private_key, ciphertext);
		}
	}
	else {
		print_usage("");
		return 1;
	}

	OQS_MEM_secure_free(input_adata, input_adata_len);
	return 0;
}