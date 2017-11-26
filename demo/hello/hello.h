#ifndef BZZVFS_H_
#define BZZVFS_H_
#define BZZVFS_MAXPATHNAME 512
#define BZZVFS_ID "bzz"
#define BZZVFS_SECSIZE 4096

typedef struct bzzFile{
	sqlite3_file base;
	int fd;
} bzzFile;

extern int bzzvfs_register();
extern int bzzvfs_open(const char*);

#endif //BZZVFS_H_
