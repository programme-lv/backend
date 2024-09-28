#include <iostream>
#include <cstdio>
#include <cmath>
#include <algorithm>
#include <stdlib.h>
using namespace std;

#define MAX (60*60*12)

int mas[MAX];

int normalize(int a)
{
	if (a < 0)
	{
		a += 60;
	}

	if (a > 30)
	{
		a = 60 - a;
	}

	return a;
}

int max(int a, int b)
{
	if (a > b)
	{
		return a;
	}
	else
	{
		return b;
	}
}

int min(int a, int b)
{
	if (a < b)
	{
		return a;
	}
	else
	{
		return b;
	}
}

int ask(int d)
{
	cout << "0 " << d << endl;

    int x;

	cin >> x;

	return x;
}

int nextdelta;

int fun(int poz[MAX], int g, int depth)
{
    int viens[21];

	viens[0] = 935;
	viens[1] = 990;
	viens[2] = 990;
	viens[3] = 7044;
	viens[4] = 1052;
	viens[5] = 994;
	viens[6] = 8741;
	viens[7] = 4045;
	viens[8] = 10456;
	viens[9] = 3869;
	viens[10] = 2526;
	viens[11] = 2499;
	viens[12] = 20949;
	viens[13] = 3638;
	viens[14] = 5440;
	viens[15] = 5457;
	viens[16] = 2130;
	viens[17] = 3391;
	viens[18] = 577;
	viens[19] = 970;
	viens[20] = 334;

	int delta = 0;
	int sk = 1000000;

	if (depth == 1)
	{
		delta = nextdelta;
	}
	else if (depth > 0)
	{
		// find delta
		for (int d = 1; d < MAX; d++)
		{
			int sad[21];
			
			for (int i = 0; i < 21; i++)
			{
			    sad[i] = 0;
			}

			for (int i = 0; i < g; i++)
			{
				sad[mas[(poz[i] + d) % (MAX)]]++;
			}

			int liel = 0;

			for (int i = 0; i <= 20; i++)
			{
				liel = max(liel, sad[i]);
			}

			if (liel < sk)
			{
				delta = d;
				sk = liel;
			}
		}
	}

//		Random r = new Random();
//		delta = r.nextInt(720);

//    delta = rand() % 720;

	int ans = ask(delta);
	
	if (depth == 0)
	{
		nextdelta = viens[ans];
	}
	
	depth++;

	int in[MAX];
	int ing = 0;

	for (int i = 0; i < g; i++)
	{
		if (mas[(poz[i]+delta) % (MAX)] == ans)
		{
			in[ing] = poz[i];
			ing++;
		}
	}
	
	if (ing > 1)
	{
		return fun(in, ing, depth);
	}
	else if (ing == 1)
	{
		return in[0];
	}
	else
	{
		return -1;
	}

}



int solve()
{
    srand (time(NULL));
    
	for (int i = 0; i < MAX; i++)
	{
			int s = i % 60;
			int m = (i / 60) % 60;
			int h = (i / 60) / 12;

			int t1 = normalize(s-m);
			int t2 = normalize(m-h);
			int t3 = normalize(h-s);

			int l = min(t1,min(t2,t3));

			mas[i] = l;
	}
	
	cout << endl;

	int inmas[MAX];
	int l = MAX;

	for (int i = 0; i < MAX; i++)
	{
		inmas[i] = i;
	}

	return fun(inmas, l, 0);
}



int main()
{
    int an = solve();
    cout << "1 " <<  an << endl;
}
