#define BZZVFS_MAXPATHNAME 512
#define BZZVFS_ID "bzz\0"
#define BZZVFS_SECSIZE 4096

typedef struct bzzFile bzzFile;
struct bzzFile{
	sqlite3_file base;
};

extern int bzzvfs_register();
extern int bzzvfs_open(const char*);

