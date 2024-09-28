#include <iostream>
#include <cstdio>
#include <cmath>
#include <algorithm>
#include <stdlib.h>
using namespace std;

int mas[720];

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

int ask(int d)
{
	cout << "0 " << d << endl;

    int x;

	cin >> x;

	return x;
}


int fun(int poz[720], int g, int depth)
{
	int delta = 0;
	int sk = 1000000;

	if (depth > 0)
	{
		// find delta
		for (int d = 1; d < 720; d++)
		{
			int sad[31];
			
			for (int i = 0; i < 31; i++)
			{
			    sad[i] = 0;
			}

			for (int i = 0; i < g; i++)
			{
				sad[mas[(poz[i] + d) % 720]]++;
			}

			int liel = 0;

			for (int i = 0; i <= 30; i++)
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

    delta = rand() % 720;

	int ans = ask(delta);

	depth++;

	int in[720];
	int ing = 0;

	for (int i = 0; i < g; i++)
	{
		if (mas[(poz[i]+delta) % 720] == ans)
		{
		    //cout << poz[i] << " ";
			in[ing] = poz[i];
			ing++;
		}
	}
	
	//cout << endl;
	//cout << "ing "  << ing << endl;

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
    
	for (int i = 0; i < 720; i++)
	{
		int m = i % 60;
		int h = i / 12;

		mas[i] = normalize(m-h);
	}

	int inmas[720];
	int l = 720;

	for (int i = 0; i < 720; i++)
	{
		inmas[i] = i;
	}

	return fun(inmas, l, 0);
}



int main()
{
    int an = solve();
    //cout << "a" << endl;
    cout << "1 " <<  an << endl;
    //cout << "z" << endl;
}
