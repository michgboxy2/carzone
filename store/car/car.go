package car

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/michgboxy2/carzone/models"
	"go.opentelemetry.io/otel"
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) Store {
	return Store{db: db}
}

func (s Store) GetCarById(ctx context.Context, id string) (models.Car, error) {
	tracer := otel.Tracer("CarStore")

	ctx, span := tracer.Start(ctx, "GetCarById-Store")

	defer span.End()

	var car models.Car

	// Prepare the SQL query to select the car and its engine details by car ID
	query := `
    SELECT 
        c.id, c.name, c.year, c.brand, c.fuel_type, e.engine_id,
        e.displacement, e.no_of_cylinders, e.car_range, 
        c.price, c.created_at, c.updated_at 
    FROM 
        cars c 
    LEFT JOIN 
        engines e ON c.engine_id = e.engine_id 
    WHERE 
        c.id = $1`

	// Execute the query
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&car.ID,
		&car.Name,
		&car.Year,
		&car.Brand,
		&car.FuelType,
		&car.Engine.EngineID,
		&car.Engine.Displacement,
		&car.Engine.NoOfCylinders,
		&car.Engine.CarRange,
		&car.Price,
		&car.CreatedAt,
		&car.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return car, nil
		}
		return car, err
	}

	return car, nil
}

func (s Store) GetCarByBrand(ctx context.Context, brand string, isEngine bool) ([]models.Car, error) {
	tracer := otel.Tracer("CarStore")

	ctx, span := tracer.Start(ctx, "GetCarByBrand-Store")

	defer span.End()

	var cars []models.Car

	// Prepare the SQL query to select cars by brand
	query := `
		SELECT 
			c.id, c.name, c.year, c.brand, c.fuel_type, 
			c.price, c.created_at, c.updated_at`

	// If isEngine is true, include engine details in the query
	if isEngine {
		query += `,
			e.engine_id, e.displacement, e.no_of_cylinders, e.car_range 
		FROM 
			cars c 
		LEFT JOIN 
			engine e ON c.engine_id = e.engine_id 
		WHERE 
			c.brand = ?`
	} else {
		query += ` 
		FROM 
			cars c 
		WHERE 
			c.brand = ?`
	}

	// Execute the query
	rows, err := s.db.QueryContext(ctx, query, brand)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Iterate through the result set
	for rows.Next() {
		var car models.Car
		if isEngine {
			err = rows.Scan(
				&car.ID,
				&car.Name,
				&car.Year,
				&car.Brand,
				&car.FuelType,
				&car.Price,
				&car.CreatedAt,
				&car.UpdatedAt,
				&car.Engine.EngineID,
				&car.Engine.Displacement,
				&car.Engine.NoOfCylinders,
				&car.Engine.CarRange,
			)
		} else {
			err = rows.Scan(
				&car.ID,
				&car.Name,
				&car.Year,
				&car.Brand,
				&car.FuelType,
				&car.Price,
				&car.CreatedAt,
				&car.UpdatedAt,
			)
		}

		if err != nil {
			return nil, err
		}
		cars = append(cars, car)
	}

	// Check for any errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return cars, nil
}

func (s Store) CreateCar(ctx context.Context, carReq *models.CarRequest) (models.Car, error) {
	tracer := otel.Tracer("CarStore")

	ctx, span := tracer.Start(ctx, "CreateCar-Store")

	defer span.End()

	var car models.Car
	var engineID uuid.UUID

	err := s.db.QueryRowContext(ctx, "SELECT id from engine where id=$1", carReq.Engine.EngineID).Scan(&engineID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return car, errors.New("engine_id does not exist in the engine table")
		}

		return car, err
	}

	//Begin Transaction
	tx, err := s.db.BeginTx(ctx, nil)

	if err != nil {
		return car, err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	query := `
	INSERT INTO cars (id, name, year, brand, fuel_type, engine_id, price, created_at, updated_at) 
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	// Get the current time for created_at and updated_at
	now := time.Now()

	// Execute the insert query
	result, err := tx.ExecContext(ctx, query, uuid.New(), carReq.Name, carReq.Year, carReq.Brand, carReq.FuelType, carReq.Engine.EngineID, carReq.Price, now, now)
	if err != nil {
		tx.Rollback()
		return car, err
	}

	// Get the last inserted ID
	carID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return car, err
	}

	// Set the car fields
	car.ID = uuid.New() // Assuming you want to generate a new UUID for the car
	car.Name = carReq.Name
	car.Year = carReq.Year
	car.Brand = carReq.Brand
	car.FuelType = carReq.FuelType
	car.Engine = carReq.Engine
	car.Price = carReq.Price
	car.CreatedAt = now
	car.UpdatedAt = now

	// Set the ID of the car
	car.ID = uuid.Must(uuid.Parse(fmt.Sprintf("%d", carID))) // Convert the last inserted ID to UUID if necessary

	return car, nil

}

func (s Store) UpdateCar(ctx context.Context, id uuid.UUID, carReq *models.CarRequest) (models.Car, error) {
	tracer := otel.Tracer("CarStore")

	ctx, span := tracer.Start(ctx, "UpdateCar-Store")

	defer span.End()

	var updatedCar models.Car

	// Begin Transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return updatedCar, err
	}

	defer func() {
		if err != nil {
			tx.Rollback() // Rollback on error
			return
		}
		err = tx.Commit() // Commit if no error
	}()

	// Prepare the SQL query to update the car
	query := `
		UPDATE cars 
		SET name = ?, year = ?, brand = ?, fuel_type = ?, price = ?, updated_at = ? 
		WHERE id = ?`

	// Get the current time for updated_at
	now := time.Now()

	// Execute the update query
	_, err = tx.ExecContext(ctx, query, carReq.Name, carReq.Year, carReq.Brand, carReq.FuelType, carReq.Price, now, id)
	if err != nil {
		return updatedCar, err // Return error if the update fails
	}

	// Set the updated car fields
	updatedCar.ID = id // Assuming id is a string that matches the car's ID
	updatedCar.Name = carReq.Name
	updatedCar.Year = carReq.Year
	updatedCar.Brand = carReq.Brand
	updatedCar.FuelType = carReq.FuelType
	updatedCar.Price = carReq.Price
	updatedCar.UpdatedAt = now

	return updatedCar, nil
}

func (s Store) DeleteCar(ctx context.Context, id string) (models.Car, error) {
	tracer := otel.Tracer("CarStore")

	ctx, span := tracer.Start(ctx, "DeleteCar-Store")

	defer span.End()

	var deletedCar models.Car

	// Begin Transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return deletedCar, err
	}

	defer func() {
		if err != nil {
			tx.Rollback() // Rollback on error
			return
		}
		err = tx.Commit() // Commit if no error
	}()

	// Prepare the SQL query to select the car before deletion
	selectQuery := `SELECT id, name, year, brand, fuel_type, price, created_at, updated_at FROM cars WHERE id = ?`
	err = tx.QueryRowContext(ctx, selectQuery, id).Scan(
		&deletedCar.ID,
		&deletedCar.Name,
		&deletedCar.Year,
		&deletedCar.Brand,
		&deletedCar.FuelType,
		&deletedCar.Price,
		&deletedCar.CreatedAt,
		&deletedCar.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return deletedCar, errors.New("car not found")
		}
		return deletedCar, err
	}

	// Prepare the SQL query to delete the car
	deleteQuery := `DELETE FROM cars WHERE id = ?`
	result, err := tx.ExecContext(ctx, deleteQuery, id)
	if err != nil {
		return deletedCar, err // Return error if the deletion fails
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		return deletedCar, err
	}

	if rowsAffected == 0 {
		return models.Car{}, errors.New("no rows were deleted")
	}

	return deletedCar, nil

}
