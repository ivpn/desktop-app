#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <dirent.h>
#include <limits.h>
#include <unistd.h>
#include <sys/stat.h>
#include <syslog.h>

#define IVPN_APP "/Applications/IVPN.app"
#define AGENT_APP "/Applications/IVPN.app/Contents/MacOS/IVPN Agent"

// TEAM_IDENTIFIER should be passed by compiler
//  Makefile example: cc -D TEAM_IDENTIFIER='"${SIGN_CERT}"' ...
//#define TEAM_IDENTIFIER "XXXXXXXXXX"

int is_safe_dir(char *dir) {
    DIR *dirp = opendir(dir);
    if(dirp == NULL)
        return 0;

    int is_safe = 0;
    struct dirent* entry;
    struct stat statBuf;
    char path[PATH_MAX];
    char *name;

    while((entry = readdir(dirp)) != NULL) {
        if(!strcmp(entry->d_name, ".") || !strcmp(entry->d_name, ".."))
            continue;

        snprintf(path, PATH_MAX, "%s/%s", dir, entry->d_name);

        if(stat(path, &statBuf))
            goto unsafe;

        if(statBuf.st_uid != 0 || statBuf.st_gid != 0) {
            printf("[helper] unsafe: %s not owned by root:wheel\n", path);
            goto unsafe;
        }

        // TODO: It looks like there is no sense checking access rights for each file in the directory. 
        //       Consider removing this check and only check the directory itself.
        if(statBuf.st_mode & S_IWOTH) {
            printf("[helper] unsafe: %s must not be writable by others\n", path);
            goto unsafe;
        }

        if(entry->d_type & DT_DIR) {
            if(!is_safe_dir(path))
                goto unsafe;
        }
    }

    is_safe = 1;

unsafe:
    closedir(dirp);

    return is_safe;
}

int main(int argc, char **argv)
{
    syslog(LOG_ALERT, "[helper] Start");
    puts("[helper] Start");

    if (!is_safe_dir("/Applications/IVPN.app"))
    {
        syslog(LOG_ALERT, "[helper] IVPN Agent seems not to have the correct(root) privileges.");
        puts("[helper] IVPN Agent seems not to have the correct(root) privileges.");
        return 1;
    }

    syslog(LOG_ALERT, "[helper] Launching:" AGENT_APP);
    puts("[helper] Launching:" AGENT_APP);

    // the second argument is 'arg0' - by convention, should point to the file name associated with the file being executed.
    execl( AGENT_APP, "IVPN Agent", NULL);

    syslog(LOG_ALERT, "[helper] Stop");
    puts("[helper] Stop");
    return 0;
}
