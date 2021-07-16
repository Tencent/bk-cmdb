const mapping = {
  $eq: 'equal',
  $neq: 'not_equal',
  $in: 'in',
  $nin: 'not_in',
  $lt: 'less',
  $lte: 'less_or_equal',
  $gt: 'greater',
  $gte: 'greater_equal',
  $range: 'between',
  $nrange: 'not_between',
  $regex: 'contains'
}

export default operator => mapping[operator]
