Ievaddatu pirmajā rindā dotas trīs naturālu skaitļu – laukuma rindu skaits $N$ $(2 \leq N)$, laukuma kolonnu skaits $M$ $(2 \leq M)$ un KP malas garums rūtiņās $K$ $(1 \leq K \leq \min(N, M))$. Tiek garantēts, ka $N \cdot M \leq 10^6$. Starp katriem diviem blakus skaitļiem ievaddatos ir tukšumzīme.

Nākamajās $N$ ievaddatu rindās dots laukuma apraksts. Katrā rindā ir tieši $M$ simboli un katram $i$ $(1 \leq i \leq N)$ un $j$ $(1 \leq j \leq M)$ simbols ievaddatu $(i + 1)$-ās rindas $j$-tajā kolonnā atbilst laukuma $i$-tās rindas $j$-tās kolonnas rūtiņas saturam un var būt:

- `.` - parasta rūtiņa
- `X` - bīstama rūtiņa
- `A` - KP sākuma atrašanās vietas kreisā augšējā stūra rūtiņa. Šī vienmēr ir parasta rūtiņa un uzdota korekti – t.i., KP pilnībā ietilpst laukumā.
- `B` - īpašā rūtiņa. Šī vienmēr ir parasta rūtiņa.
