#include "stdio.h"
#include <algorithm>
using namespace std;

const long MAKSIS = 100000;

long a[MAKSIS];
long n, m, delta, starp, x;

int main() {

scanf("%ld",&n);
for (long i = 0; i<n; i++) scanf("%ld",&a[i]);

sort(a,a+n);
m = (n + 1) / 2;

// Noskaidro mazāko starpību
delta = n - m;
starp = a[n-1] - a[m-1];

for (long i=n-2; i>=m; i--) {
	x = a[i] - a[i-delta];
	if (x<starp) starp = x;	
}

printf("%ld\n",starp);
printf("%ld %ld",a[m-1],a[n-1]);
for (long i=n-2; i>=m; i--) {
	printf(" %ld %ld",a[i-delta],a[i]);
}
if (n % 2 == 1) printf(" %ld",a[0]);
printf("\n");

return 0;
}
