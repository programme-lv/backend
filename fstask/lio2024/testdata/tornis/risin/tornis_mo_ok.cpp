#include <iostream>

// "Torņi" - Valsts 2024
//
// (c) Mārtiņš Opmanis, 2024

using namespace std;

const int MAKSIS = 1000005;

typedef struct { int maz; int liel; int iepr; int nak; } raksts;
typedef struct { int p; int o; } paris;

raksts a[MAKSIS];
paris  p[MAKSIS];

int main ()
{
	raksts *r1;
	raksts *r2;
	raksts *rb;
	paris px;
	int p_kur = 0;
	
	
	r1 = &a[0];
	r1 -> maz = -1;
	r1 -> liel = -2;
	r1 -> iepr = -1;
	r1 -> nak = 1;
	
	int N;
	cin >> N;	
	
	for (int i=1; i<=N; i++) {
		r1 = &a[i];
		cin >> r1 -> maz;
		r1 -> liel = r1 -> maz;
		r1 -> iepr = i-1;
		r1 -> nak = i+1;
	}
	
	rb = &a[N+1];
	rb -> maz = -1;
	rb -> liel = -2;
	rb -> iepr = N;
	rb -> nak = -1;
	
	r1 = &a[1];
	while ( r1 != rb) {
		
//		cout << r1->maz << " " << r1->liel << " " << r1->iepr << " " << r1->nak << "\n";
		
		r2 = &a[r1 -> iepr];
		
		if ( r1 -> liel + 1 == r2 -> maz ) {
			r1 -> liel = r2 -> liel;
			a[r2 -> iepr].nak = r2 -> nak;
			r1 -> iepr = r2 -> iepr;
			px.p = r1 -> maz;
			px.o = r2 -> maz;
			p[p_kur] = px;
//			cout << "p[" << p_kur << "] " << px.p << " " << px.o << "\n";
			p_kur++;
		}
		else if ( r2 -> liel + 1 == r1 -> maz ) {
			r2 -> liel = r1 -> liel;
			a[r1 -> nak].iepr = r1 -> iepr;
			r2 -> nak = r1 -> nak;
			px.p = r2 -> maz;
			px.o = r1 -> maz;
			p[p_kur] = px;
//			cout << "p[" << p_kur << "] " << px.p << " " << px.o << "\n";
			p_kur++;
			r1 = r2;
		}
		else r1 = &a[r1 -> nak];
	}
	
	int maks_tornis = -1;
	int maks_t_ind = 0;
	int j = a[0].nak;
	while (j <= N) {
		if ( a[j].liel - a[j].maz > maks_tornis ) {
			maks_tornis = a[j].liel - a[j].maz; 
			maks_t_ind = j;
			}
		j = a[j].nak;
	}
	
	int x_maz = a[maks_t_ind].maz;
	int x_liel = a[maks_t_ind].liel;
	
	cout << maks_tornis + 1 << " " << maks_tornis << "\n";
	for (int i=0; i<p_kur; i++) {
		if ((p[i].p >= x_maz) && (p[i].o <= x_liel)) cout << p[i].p << " " << p[i].o << "\n";
	}
	
	return 0;
}