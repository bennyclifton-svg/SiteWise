"""Offline checks for the experiment's candidate and coordinate boundaries."""
import unittest
from titleblock_experiment import valid_box, joined_lines, reading_rect, selected


class ExperimentTests(unittest.TestCase):
    def test_box_rejects_bad_geometry(self):
        self.assertTrue(valid_box(dict(left=0,top=0,right=1,bottom=1)))
        for b in [dict(left=.8,right=.2,top=0,bottom=1),
                  dict(left=0,right=1,top=float('nan'),bottom=1),
                  dict(left=-.1,right=1,top=0,bottom=1),{}]:
            self.assertFalse(valid_box(b))

    def test_multiline_rotated_text_does_not_join_other_cell(self):
        def run(text,x,y,page=1):
            return {'Text':text,'Source':dict(X=x,Y=y,Width=10,Height=60,Page=page,Rotation=270)}
        a=run('LOWER LEVEL',100,300)
        b=run('AREA B',114,300)
        wrong=run('OTHER CELL',114,400)
        otherpage=run('OTHER PAGE',114,300,2)
        self.assertEqual(joined_lines([a,b,wrong,otherpage]),['LOWER LEVEL AREA B'])

    def test_selection_cannot_create_text(self):
        def record(value):return {'response':{'answers':{'title':{'choice':value}}}}
        self.assertEqual(selected(record('0'),['Exact source']), 'Exact source')
        self.assertIsNone(selected(record('none'),['Exact source']))
        for value in ['invented','-1','1']:
            with self.assertRaises(ValueError):selected(record(value),['Exact source'])

    def test_rotated_reading_coordinates(self):
        self.assertEqual(reading_rect(dict(X=100,Y=300,Width=10,Height=60,Rotation=270)),(300,100,60,10))


if __name__=='__main__':unittest.main()
