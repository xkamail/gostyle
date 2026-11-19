package exportfirst

type Store struct{}

type cache[K comparable] struct{}

func init() {}

func New() *Store { return &Store{} }

func (s *Store) Get() {}

func helper() {}

func (s *Store) Put() {} // want `exported Put is declared below unexported helper \(line 13\)`

func (c *cache[K]) Len() int { return 0 }

func Open() {} // want `exported Open is declared below unexported helper \(line 13\)`
