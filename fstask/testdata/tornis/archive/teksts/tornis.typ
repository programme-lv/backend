#import "/template/template.typ": *

#let (
    conf,
    print_example, 
    print_example_raw,
    subtask_restriction_table, 
    restrictions_and_requirements,
    contest,
    task,
) = prepare_task_document(
    contest_yaml: "/valsts_jaun_d2.yaml",
    task_codename: "Tornis",
)

#show: doc => conf(
    doc,
)

Valters pēta piramīdas, kas sastāv no $N$ atšķirīga izmēra ripām, kas sanumurētas no mazākās līdz lielākajai pēc kārtas ar skaitļiem no $1$ līdz $N$. Par _sakārtotu torni_ Valters sauc gan atsevišķu ripu, gan ripu torni, ko veido viena uz otras saliktas ripas, kur ripu numuri ir pēc kārtas. Sākumā visas ripas Valters saliek uz galda patvaļīgā secībā vienā rindā. Katrā gājienā Valters var uzlikt vienu sakārtotu torni uz blakus sakārtota torņa, ja viena torņa apakšējās ripas numurs ir par vienu mazāks nekā blakus torņa augšējās ripas numurs. Valtera mērķis ir noteikt, kādu augstāko (ar vislielāko ripu skaitu) sakārtoto torni iespējams izveidot dotajam sākotnējam ripu sakārtojumam.

Sakārotu torni, kura augšējā ripa ir $l$, bet apakšējā -- $r$, apzīmēsim kā $l\-r$. Apskatot piemēru, ja ir sešas ripas, kuru sākuma secība ir $3, 2, 4, 1, 6, 5$ (skat. @piem(a) att.), tad vispirms var uzlikt torni $2\-2$ uz torņa $3\-3$ (@piem(b) att.), tad šo sakārtoto torni $2\-3$ uz torņa $4\-4$ (@piem(c) att.), pēc tam var izveidot sakārtotu torni, uzliekot $5\-5$ uz $6\-6$ (@piem(d) att.), uz sakārtota torņa $2\-4$ uzlikt torni $1\-1$ (@piem(e) att.) un, visbeidzot, uzlikt sakārtoto torni $1\-4$ uz sakārtota torņa $5\-6$ (@piem(f) att.). Tādējādi šajā gadījumā visas ripas var salikt vienā sakārtotā tornī, kura augstums ir $6$.

#figure(
    caption: [Ripu sakārtošanas tornī piemērs],
    image("tornis.png", height: 40em)
)<piem>

Uzrakstiet datorprogrammu, kas, ievadītai ripu sākotnējai secībai, nosaka lielāko iespējamo sakārtota torņa augstumu un izvada īsāko gājienu virkni, kuras rezultātā šāda augstuma tornis tiek izveidots!

#pagebreak()

== Ievaddati

Pirmajā rindā dots ripu skaits -- naturāls skaitlis $N (N ≤ 5 dot 10^5)$. 

Otrajā rindā doti $N$ atšķirīgi naturāli skaitļi $a_i (1<=a_i<=N)$ -- ripu lielumi, kas atdalīti ar tukšumzīmēm.

== Izvaddati

Izvaddatu pirmajā rindā jābūt diviem, ar tukšumzīmi atdalītiem, naturāliem skaitļiem $M$ un $K$ -- augstākajam sakārtotā torņa augstumam, kādu iespējams izveidot, un mazākajam gājienu skaitam, lai tik augstu torni izveidotu. Nākamajās $K$ izvaddatu rindās jāapraksta izdarīto gājienu virkne, pa vienam katrā rindā. Katra gājiena aprakstu veido divi, ar tukšumzīmi atdalīti, naturāli skaitļi $x$ un $y$, kas nozīmē, ka sakārtots tornis, kuram augšā ir ripa $x$, tiek uzlikts uz blakus sakārtota torņa, kuram augšā ir ripa~$y$. Aprakstīto gājienu virknei ir jāizveido vismaz viens sakārtots tornis augstumā $M$.

Ja augstāko sakārtoto torni iespējams iegūt dažādos veidos, nepieciešams izvadīt jebkuru vienu derīgu gājienu secību.

== Ierobežojumi un prasības

#restrictions_and_requirements()

== Piemēri

#grid(
    columns: (50%, 50%),
    gutter: 1em,
    [
        #print_example(
            input_width: 35%,
            comment_width: 43%,
            output_width: 22%,
            comment: "Atbilst piemēram uzdevuma tekstā. Ir iespējama arī cita derīga gājienu secība.",
            "tornis.i00a",
        )
    ],
    [
         #print_example(
            input_width: 35%,
            comment_width: 43%,
            output_width: 22%,
            comment: "Nevar izdarīt nevienu gājienu -- augstākais sakārtotais tornis ir vienu ripu augsts.",
            "tornis.i00b",
        )
    ],
)

#grid(
    columns: (80%),
    gutter: 1em,
    [
        #print_example(
            input_width: 36%,
            output_width: 26%,
            comment: "Derētu arī gājienu virkne \n2 3\n2 4",
            "tornis.i00c",
        )
    ],
)

== 1. apakšuzdevuma testu ievaddati

#grid(columns: 3, gutter: 1em, 
    [
         #print_example(
             "tornis.i01a", 
             output: false,
         )
    ],
    [
         #print_example(
             "tornis.i01b", 
             output: false,
         )
    ],
    [
         #print_example(
             "tornis.i01c", 
            output: false,
         )
    ]
)
    

    


#block(
    breakable: false,
[
== Apakšuzdevumi un to vērtēšana

#subtask_restriction_table((
    none,
  [ Uzdevuma tekstā dotie trīs testi ],
  [ Ripas ir novietotas rindā augošā secībā pēc to lielumiem, jeb $a_i < a_(i+1)$ visiem $1<=i<=n-1$ ],
  [ $N <= 10$ ],
  [ $M = N$ jeb augstākais tornis sastāvēs no visām $N$ ripām ],
  [ $N <= 3000$ ],
  [ Bez papildu ierobežojumiem ]
))
]
)